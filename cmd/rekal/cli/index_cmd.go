package cli

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	var includeAll bool
	var include []string
	var noLocal bool

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Rebuild the index DB from the data DB",
		Long: `Drop and rebuild the index DB (.rekal/index.db) from the data DB.

The index is local-only and never synced. It contains:
  - Full-text search index (BM25) over conversation turns
  - LSA vector embeddings for lightweight semantic similarity
  - Nomic deep semantic embeddings (on supported platforms)
  - Session facets (author, branch, actor, counts) for fast filtering
  - File co-occurrence graph
  - Tool call indexes

Rebuild when the index is out of date or after importing new data.
'rekal sync' rebuilds the index automatically.

Cross-repo local import (--include-all / --include / --no-local) folds your
own Claude Code history from other repos and shell sessions on this machine
into recall. These sessions are stored in the index only, never in the data
DB, so they can be recalled locally but can never be pushed to the team.
The choice is remembered: plain 'rekal index' and 'rekal sync' keep whatever
you last set. Default is off.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			if err := applyLocalPrefFlags(cmd, gitRoot, includeAll, include, noLocal); err != nil {
				return err
			}

			return runIndex(cmd, gitRoot)
		},
	}

	cmd.Flags().BoolVar(&includeAll, "include-all", false,
		"import every local Claude Code session on this machine (all repos + shell) into recall")
	cmd.Flags().StringArrayVar(&include, "include", nil,
		"import local sessions for specific repo path(s) into recall (repeatable)")
	cmd.Flags().BoolVar(&noLocal, "no-local", false,
		"stop importing local cross-repo sessions (clears the remembered choice)")

	return cmd
}

// applyLocalPrefFlags translates the cross-repo import flags into a persisted
// preference before the rebuild. Only one of the flags may be set at a time.
// When none is set, the remembered preference is left untouched so a plain
// rebuild honors the last choice.
func applyLocalPrefFlags(cmd *cobra.Command, gitRoot string, includeAll bool, include []string, noLocal bool) error {
	set := 0
	if includeAll {
		set++
	}
	if len(include) > 0 {
		set++
	}
	if noLocal {
		set++
	}
	if set == 0 {
		return nil
	}
	if set > 1 {
		return fmt.Errorf("--include-all, --include, and --no-local are mutually exclusive")
	}

	cfg, err := readConfig(gitRoot)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	w := cmd.ErrOrStderr()
	switch {
	case noLocal:
		cfg.LocalImport = localPref{}
		fmt.Fprintln(w, "rekal: cross-repo local import off")
	case includeAll:
		cfg.LocalImport = localPref{All: true}
		fmt.Fprintln(w, "rekal: cross-repo local import on (all local sessions)")
	default:
		// Normalize to absolute so a repo path resolves to the same project
		// dir the current repo would (the sanitizer keys off the absolute cwd).
		abs := make([]string, 0, len(include))
		for _, p := range include {
			if a, aerr := filepath.Abs(p); aerr == nil {
				abs = append(abs, a)
			} else {
				abs = append(abs, p)
			}
		}
		cfg.LocalImport = localPref{Repos: abs}
		fmt.Fprintf(w, "rekal: cross-repo local import on (%d repo path(s))\n", len(abs))
	}

	if err := writeConfig(gitRoot, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// runIndex rebuilds the index DB into a temporary file and only replaces
// the live index.db with it — via an atomic rename — once the entire
// rebuild has succeeded. index.db is always derived and rebuildable in
// principle, but the old drop-tables-then-repopulate-in-place approach
// meant a kill mid-rebuild (OOM, disk full, Ctrl-C) could leave the live
// index.db with some tables dropped and not yet repopulated — recall/query
// would then silently run against a partially empty index with no signal
// anything was wrong. Building into a fresh file means any failure simply
// leaves the previous, fully-built index.db in place untouched.
func runIndex(cmd *cobra.Command, gitRoot string) error {
	w := cmd.ErrOrStderr()

	livePath := db.IndexPath(gitRoot)
	tmpPath := livePath + ".rebuilding"

	// Clear any leftover temp file (and its WAL, if DuckDB left one behind)
	// from a previous interrupted rebuild before starting a fresh one.
	_ = os.Remove(tmpPath)
	_ = os.Remove(tmpPath + ".wal")

	indexDB, err := db.OpenIndexAt(tmpPath)
	if err != nil {
		return fmt.Errorf("open index db: %w", err)
	}
	// On any failure below, close and discard the half-built temp file —
	// the live index.db (if any) is never touched until the rename at the
	// end. On success this Close is a harmless no-op double-close guard
	// (rebuildOK short-circuits the removal).
	rebuildOK := false
	defer func() {
		indexDB.Close()
		if !rebuildOK {
			_ = os.Remove(tmpPath)
		}
	}()

	// Load FTS extension.
	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Create schema. Building into a brand-new file means there's nothing
	// to drop first — DropIndexTables existed only to support the old
	// in-place rebuild.
	if err := db.InitIndexSchema(indexDB); err != nil {
		return fmt.Errorf("create index schema: %w", err)
	}

	// Populate from data DB.
	fmt.Fprintln(w, "populating index from data db...")
	if err := db.PopulateIndex(indexDB, gitRoot); err != nil {
		return fmt.Errorf("populate index: %w", err)
	}

	// Cross-repo local import (index-only; never touches data.db). Honors the
	// remembered preference — off by default.
	cfg, err := readConfig(gitRoot)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	pref := cfg.LocalImport
	if pref.enabled() {
		fmt.Fprintln(w, "importing local cross-repo sessions...")
		localSessions, localProjects, ierr := importLocalSessions(indexDB, gitRoot, pref, w)
		if ierr != nil {
			return fmt.Errorf("import local sessions: %w", ierr)
		}
		fmt.Fprintf(w, "imported %d local session(s) from %d project(s)\n", localSessions, localProjects)
	}

	// Count what we indexed.
	var sessionCount, turnCount int
	if err := indexDB.QueryRow("SELECT count(*) FROM session_facets").Scan(&sessionCount); err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&turnCount); err != nil {
		return fmt.Errorf("count turns: %w", err)
	}

	// Create FTS index (only if there are turns).
	if turnCount > 0 {
		fmt.Fprintln(w, "creating full-text search index...")
		if err := db.CreateFTSIndex(indexDB); err != nil {
			return fmt.Errorf("create fts index: %w", err)
		}
	}

	// LSA pass.
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
			fmt.Fprintf(w, "stored %d LSA embeddings (%d dimensions)\n", len(vectors), embeddingDim)

			// Cache the query-projection state so recall doesn't refit the
			// whole model on every call — non-fatal, recall falls back to
			// rebuilding from raw content if this is missing.
			if err := search.StoreLSAProjection(indexDB, model); err != nil {
				fmt.Fprintf(w, "warning: caching LSA projection failed: %v\n", err)
			}
		}

		// Nomic pass (non-fatal).
		if err := buildNomicEmbeddings(indexDB, sessionContent, w, gitRoot); err != nil {
			fmt.Fprintf(w, "warning: nomic embeddings skipped: %v\n", err)
		}
	}

	// Write index state.
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

	// Everything succeeded — close the temp DB (DuckDB checkpoints/flushes
	// on close) and atomically replace the live index.db with it. Rename
	// within the same directory is atomic on the filesystems rekal targets
	// (POSIX rename(2)): a reader opening index.db either sees the old file
	// or the fully-built new one, never a partial file.
	if err := indexDB.Close(); err != nil {
		return fmt.Errorf("close rebuilt index: %w", err)
	}
	if err := os.Rename(tmpPath, livePath); err != nil {
		return fmt.Errorf("replace index db: %w", err)
	}
	rebuildOK = true

	fmt.Fprintf(w, "index rebuilt: %d sessions, %d turns\n", sessionCount, turnCount)
	return nil
}

// buildNomicEmbeddings generates nomic-embed-text embeddings for all sessions
// and stores them in the index DB. Non-fatal: returns error on any failure.
func buildNomicEmbeddings(indexDB *sql.DB, sessionContent map[string]string, w io.Writer, gitRoot string) error {
	if !nomic.Supported() {
		return nil
	}

	fmt.Fprintln(w, "building nomic deep semantic embeddings...")
	client, err := nomic.NewClient(gitRoot)
	if err != nil {
		return err
	}
	defer client.Close()

	vectors, err := client.EmbedSessions(sessionContent)
	if err != nil {
		return err
	}

	if err := db.StoreEmbeddings(indexDB, vectors, nomic.ModelName); err != nil {
		return err
	}
	fmt.Fprintf(w, "stored %d nomic embeddings (%d dimensions)\n", len(vectors), nomic.EmbedDim)
	return nil
}
