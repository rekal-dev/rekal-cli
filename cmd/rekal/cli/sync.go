package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var selfOnly bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync team context from remote rekal branches",
		Long: `Fetch team session data from remote rekal branches and import into the local database.

This is a manual, deliberate operation — it is NOT automated via git hooks.
Run it when you want to pull your team's session history before starting work.

Imported data includes conversation turns, tool calls, and file change metadata
from your teammates' AI coding sessions. Imported checkpoints are marked as
exported so they are never re-pushed to your own branch.

By default, fetches all rekal/* branches (whole team). Use --self to fetch
only your own rekal branch — useful when syncing across your own machines
(e.g. pulling context from your work laptop to your home machine).

Typical usage:
  Developer:  Run 'rekal sync' at the start of the day
  Agent:      Run 'rekal sync' at the start of a session if team context matters
  Ad-hoc:     Run 'rekal sync --self' to pull your own data from another machine`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}
			if err := EnsureInitDone(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			if selfOnly {
				return runSyncSelf(cmd, gitRoot)
			}
			return runSyncTeam(cmd, gitRoot)
		},
	}

	cmd.Flags().BoolVar(&selfOnly, "self", false, "Only fetch your own rekal branch (not the whole team)")

	return cmd
}

// runSyncTeam checkpoints + pushes local data, fetches all remote rekal branches,
// and rebuilds the index from local data.db plus decoded remote wire format.
func runSyncTeam(cmd *cobra.Command, gitRoot string) error {
	w := cmd.ErrOrStderr()

	// Step 1: Checkpoint (non-fatal).
	if err := doCheckpoint(gitRoot, w); err != nil {
		fmt.Fprintf(w, "rekal: warning: checkpoint failed: %v\n", err)
	}

	// Step 2: Push (non-fatal).
	if err := doPush(gitRoot, w, false); err != nil {
		fmt.Fprintf(w, "rekal: warning: push failed: %v\n", err)
	}

	// Step 3: Fetch remote rekal refs (non-fatal).
	fmt.Fprintln(w, "fetching remote rekal branches...")
	if err := fetchRemoteRekalRefs(gitRoot); err != nil {
		fmt.Fprintf(w, "rekal: warning: fetch failed: %v\n", err)
	}

	// Step 4: List remote branches (excluding self).
	remoteBranches, err := listRemoteRekalBranches(gitRoot)
	if err != nil {
		fmt.Fprintf(w, "rekal: warning: listing remote branches failed: %v\n", err)
	}

	// Step 5: Rebuild index.
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return fmt.Errorf("open index db: %w", err)
	}
	defer indexDB.Close()

	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Clean slate.
	if err := db.DropIndexTables(indexDB); err != nil {
		return fmt.Errorf("drop index tables: %w", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		return fmt.Errorf("create index schema: %w", err)
	}

	// 5a: Populate from local data.db.
	fmt.Fprintln(w, "indexing local data...")
	if err := db.PopulateIndex(indexDB, gitRoot); err != nil {
		return fmt.Errorf("populate index: %w", err)
	}

	// Count local sessions.
	var localSessions int
	if err := indexDB.QueryRow("SELECT count(*) FROM session_facets").Scan(&localSessions); err != nil {
		return fmt.Errorf("count local sessions: %w", err)
	}

	// 5b: Import each remote branch into index.
	var remoteSessions int
	teamMembers := 0
	for _, branch := range remoteBranches {
		fmt.Fprintf(w, "importing %s...\n", branch)
		n, err := importBranchToIndex(gitRoot, indexDB, branch)
		if err != nil {
			fmt.Fprintf(w, "rekal: warning: import %s failed: %v\n", branch, err)
			continue
		}
		if n > 0 {
			remoteSessions += n
			teamMembers++
		}
	}

	// Count totals.
	var sessionCount, turnCount int
	if err := indexDB.QueryRow("SELECT count(*) FROM session_facets").Scan(&sessionCount); err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&turnCount); err != nil {
		return fmt.Errorf("count turns: %w", err)
	}

	// 5c: Create FTS index.
	if turnCount > 0 {
		fmt.Fprintln(w, "creating full-text search index...")
		if err := db.CreateFTSIndex(indexDB); err != nil {
			return fmt.Errorf("create fts index: %w", err)
		}
	}

	// 5d: LSA pass.
	embeddingDim := 0
	if sessionCount >= 2 {
		fmt.Fprintln(w, "building LSA embeddings...")
		sessionContent, err := db.QuerySessionContent(indexDB)
		if err != nil {
			return fmt.Errorf("query session content: %w", err)
		}

		model, err := lsa.Build(sessionContent, lsa.DefaultDimension)
		if err != nil {
			fmt.Fprintf(w, "warning: LSA build failed: %v\n", err)
		} else if model != nil {
			vectors := model.Vectors()
			if err := db.StoreEmbeddings(indexDB, vectors, "lsa-v1"); err != nil {
				return fmt.Errorf("store embeddings: %w", err)
			}
			embeddingDim = model.Dim
		}

		// 5d-ii: Nomic pass (non-fatal).
		if err := buildNomicEmbeddings(indexDB, sessionContent, w); err != nil {
			fmt.Fprintf(w, "warning: nomic embeddings skipped: %v\n", err)
		}
	}

	// 5e: Write index state.
	if err := db.WriteIndexState(indexDB, "session_count", strconv.Itoa(sessionCount)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "turn_count", strconv.Itoa(turnCount)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "embedding_dim", strconv.Itoa(embeddingDim)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "last_indexed_at", "now"); err != nil {
		return err
	}

	// Step 6: Summary.
	fmt.Fprintf(w, "rekal: synced — %d local sessions", localSessions)
	if remoteSessions > 0 {
		fmt.Fprintf(w, ", %d remote sessions from %d team member(s)", remoteSessions, teamMembers)
	}
	fmt.Fprintln(w)

	return nil
}

// runSyncSelf fetches the current user's remote branch, imports into data.db,
// and performs a full index rebuild.
func runSyncSelf(cmd *cobra.Command, gitRoot string) error {
	w := cmd.ErrOrStderr()
	branch := rekalBranchName()

	// Step 1: Fetch own remote branch.
	fmt.Fprintln(w, "fetching your remote branch...")
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		return fmt.Errorf("no remote 'origin' configured")
	}

	fetchCmd := exec.Command("git", "-C", gitRoot, "fetch", "origin", branch)
	fetchCmd.Stdin = nil
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch origin/%s failed: %s", branch, strings.TrimSpace(string(output)))
	}

	// Step 2: Import from remote branch into data.db.
	remoteBranch := "origin/" + branch
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data db: %w", err)
	}

	n, err := importBranch(gitRoot, dataDB, remoteBranch)
	dataDB.Close()
	if err != nil {
		return fmt.Errorf("import from %s: %w", remoteBranch, err)
	}
	fmt.Fprintf(w, "rekal: imported %d session(s) from %s\n", n, remoteBranch)

	// Step 3: Full index rebuild.
	return runIndex(cmd, gitRoot)
}
