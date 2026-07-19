package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/ids"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/scrub"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
	"github.com/spf13/cobra"
)

func newCheckpointCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkpoint",
		Short: "Capture the current session after a commit",
		Long: `Snapshot the active AI session into the local data DB.

Reads session transcript files (conversation turns, tool calls, file changes)
from the agent's session directory, deduplicates by content hash, and inserts
into .rekal/data.db. Each checkpoint is linked to the current HEAD commit and
records which files were changed.

Normally runs automatically via the post-commit hook installed by 'rekal init'.
Run manually to capture a session without committing.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			return runCheckpoint(cmd, gitRoot)
		},
	}
}

func runCheckpoint(cmd *cobra.Command, gitRoot string) error {
	return doCheckpoint(gitRoot, cmd.ErrOrStderr())
}

// doCheckpoint captures the current session after a commit.
// Extracted so sync can call it without a cobra.Command.
func doCheckpoint(gitRoot string, w io.Writer) error {
	// Open data DB.
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	// Run forward-only migrations for existing DBs.
	if err := db.MigrateDataSchema(dataDB); err != nil {
		return fmt.Errorf("migrate data schema: %w", err)
	}

	// Verify DB is healthy by running a simple query.
	if _, err := dataDB.Exec("SELECT 1"); err != nil {
		return fmt.Errorf("data DB is corrupt or unreadable: %w", err)
	}

	email := gitx.ConfigValue("user.email")
	newID := ids.NewULIDFunc()

	var sessionIDs []string
	// trunkOnlySessionIDs is the subset of sessionIDs with no parent — used to
	// bound synchronous embedding work (see updateIndexIncremental).
	var trunkOnlySessionIDs []string
	var inserted int
	// Collect unique relative file paths from file-modifying tool_calls across all sessions.
	toolCallPaths := make(map[string]struct{})

	// trunkSessionIDs maps trunk session file paths inserted in this run to
	// their DB session IDs, so subagent sessions (which Discover returns
	// after their trunk) can link to the parent row via parent_session_id.
	trunkSessionIDs := make(map[string]string)

	// Iterate all adapters to discover sessions from all known agents.
	for _, adapter := range session.Adapters {
		refs, err := adapter.Discover(gitRoot)
		if err != nil {
			continue
		}

		for _, ref := range refs {
			// Determine cache key for deduplication.
			cacheKey := ref.Path
			if cacheKey == "" {
				cacheKey = adapter.Name() + ":" + ref.DBID
			}

			// For file-based refs, check size+hash cache.
			var data []byte
			var hash string
			if ref.Path != "" {
				info, statErr := os.Stat(ref.Path)
				if statErr != nil {
					continue
				}

				fileData, err := os.ReadFile(ref.Path)
				if err != nil || len(fileData) == 0 {
					continue
				}
				data = fileData
				hash = sha256Hex(data)

				cachedSize, cachedHash, found, csErr := db.GetCheckpointState(dataDB, cacheKey)
				if csErr != nil {
					return fmt.Errorf("check checkpoint state: %w", csErr)
				}
				if found && cachedSize == info.Size() && cachedHash == hash {
					continue
				}
			} else {
				// DB-based ref — use DBID as hash seed for dedup.
				hash = sha256Hex([]byte(cacheKey))

				_, _, found, csErr := db.GetCheckpointState(dataDB, cacheKey)
				if csErr != nil {
					return fmt.Errorf("check checkpoint state: %w", csErr)
				}
				if found {
					continue
				}
			}

			exists, err := db.SessionExistsByHash(dataDB, hash)
			if err != nil {
				return fmt.Errorf("dedup check: %w", err)
			}
			if exists {
				if ref.Path != "" {
					info, _ := os.Stat(ref.Path)
					if info != nil {
						_ = db.UpsertCheckpointState(dataDB, cacheKey, info.Size(), hash)
					}
				} else {
					_ = db.UpsertCheckpointState(dataDB, cacheKey, 0, hash)
				}
				continue
			}

			payload, err := adapter.Parse(ref)
			if err != nil || payload == nil {
				continue
			}

			// Redact secrets and anonymize paths before any DB insertion.
			scrub.Scrub(payload)

			if len(payload.Turns) == 0 && len(payload.ToolCalls) == 0 {
				continue
			}

			// Never capture RekalBench / harness sessions into a real store
			// (BUG 6 — synthetic fixtures pollute recall).
			if session.SkipCapture(payload) {
				continue
			}

			sessionID := newID()
			capturedAt := time.Now().UTC()

			// Resolve the parent session for subagent transcripts: prefer a
			// trunk inserted in this run, then fall back to looking up the
			// trunk file's content hash from a previous capture.
			parentSessionID := ""
			if payload.ParentSessionPath != "" {
				parentSessionID = trunkSessionIDs[payload.ParentSessionPath]
				if parentSessionID == "" {
					if trunkData, rdErr := os.ReadFile(payload.ParentSessionPath); rdErr == nil && len(trunkData) > 0 {
						if id, qErr := db.QuerySessionIDByHash(dataDB, sha256Hex(trunkData)); qErr == nil {
							parentSessionID = id
						}
					}
				}
			}

			// Insert session into DuckDB.
			if err := db.InsertSessionMeta(
				dataDB, sessionID, parentSessionID, hash,
				payload.ActorType, payload.AgentID, email, payload.Branch, capturedAt.Format(time.RFC3339),
				payload.Source, db.SessionMetaFields{
					TeamName:     payload.TeamName,
					WorkflowName: payload.WorkflowName,
					AgentType:    payload.AgentType,
					Description:  payload.Description,
					SpawnDepth:   payload.SpawnDepth,
				},
			); err != nil {
				return fmt.Errorf("insert session: %w", err)
			}
			if payload.ParentSessionPath == "" && ref.Path != "" {
				trunkSessionIDs[ref.Path] = sessionID
			}
			if parentSessionID == "" {
				trunkOnlySessionIDs = append(trunkOnlySessionIDs, sessionID)
			}

			// Insert turns into DuckDB.
			for i, t := range payload.Turns {
				ts := ""
				if !t.Timestamp.IsZero() {
					ts = t.Timestamp.UTC().Format(time.RFC3339)
				}
				if err := db.InsertTurn(dataDB, newID(), sessionID, i, t.Role, t.Content, ts); err != nil {
					return fmt.Errorf("insert turn: %w", err)
				}
			}

			// Insert tool calls into DuckDB.
			for i, tc := range payload.ToolCalls {
				if err := db.InsertToolCall(dataDB, newID(), sessionID, i, tc.Tool, tc.Path, tc.CmdPrefix); err != nil {
					return fmt.Errorf("insert tool_call: %w", err)
				}
			}

			// Collect file-modifying tool_call paths for files_touched supplementation.
			for _, tc := range payload.ToolCalls {
				if tc.Path == "" {
					continue
				}
				switch tc.Tool {
				case "Write", "Edit", "NotebookEdit", "StrReplace":
				default:
					continue
				}
				rel := strings.TrimPrefix(tc.Path, gitRoot+"/")
				if rel == tc.Path {
					continue
				}
				toolCallPaths[rel] = struct{}{}
			}

			// Update checkpoint state cache.
			if ref.Path != "" {
				info, _ := os.Stat(ref.Path)
				if info != nil {
					_ = db.UpsertCheckpointState(dataDB, cacheKey, info.Size(), hash)
				}
			} else {
				_ = db.UpsertCheckpointState(dataDB, cacheKey, 0, hash)
			}

			sessionIDs = append(sessionIDs, sessionID)
			inserted++
		}
	}

	if inserted == 0 {
		return nil
	}

	// Get git state for checkpoint.
	gitSHA := gitx.HeadSHA(gitRoot)
	gitBranch := gitx.CurrentBranch(gitRoot)
	filesTouched := gitx.FilesChanged(gitRoot)

	// Generate checkpoint ULID.
	checkpointID := newID()

	// Insert checkpoint into DuckDB (exported = FALSE by default).
	now := time.Now().UTC()
	if err := db.InsertCheckpoint(dataDB, checkpointID, gitSHA, gitBranch, email, now.Format(time.RFC3339), "human", ""); err != nil {
		return fmt.Errorf("insert checkpoint: %w", err)
	}

	// Insert files_touched from git diff.
	gitTouchedSet := make(map[string]struct{})
	for _, ft := range filesTouched {
		parts := strings.SplitN(ft, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		gitTouchedSet[parts[1]] = struct{}{}
		if err := db.InsertFileTouched(dataDB, newID(), checkpointID, parts[1], parts[0]); err != nil {
			return fmt.Errorf("insert file_touched: %w", err)
		}
	}

	// Supplement files_touched with file-modifying tool_call paths not already covered by git diff.
	for p := range toolCallPaths {
		if _, exists := gitTouchedSet[p]; exists {
			continue
		}
		if err := db.InsertFileTouched(dataDB, newID(), checkpointID, p, "T"); err != nil {
			return fmt.Errorf("insert file_touched (tool_call): %w", err)
		}
	}

	// Insert checkpoint_sessions junction rows.
	for _, sid := range sessionIDs {
		if err := db.InsertCheckpointSession(dataDB, checkpointID, sid); err != nil {
			return fmt.Errorf("insert checkpoint_session: %w", err)
		}
	}

	// Incrementally update the index for newly captured sessions.
	if err := updateIndexIncremental(gitRoot, sessionIDs, trunkOnlySessionIDs, checkpointID, w); err != nil {
		// Non-fatal — index can be rebuilt later with 'rekal index'.
		fmt.Fprintf(w, "rekal: warning: incremental index update failed: %v\n", err)
	}

	fmt.Fprintf(w, "rekal: %d session(s) captured\n", inserted)
	return nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// updateIndexIncremental adds newly captured sessions to the index DB
// without a full rebuild. Handles: turns_ft, tool_calls_index, session_facets,
// files_index, and nomic embeddings. LSA is skipped (requires full corpus).
// FTS pragma_create_fts_index is not re-run — new rows in turns_ft are
// automatically indexed by DuckDB's FTS.
//
// Nomic embedding is synchronous CGO work with real per-call cost, and
// embeddableSessionIDs — not the full sessionIDs — bounds how much of it runs
// here: only trunk sessions (no parent_session_id) are embedded during
// checkpoint. A commit can discover many new subagent/workflow transcripts at
// once (recursive discovery re-scans every subagents/ directory each run);
// embedding all of them synchronously in the post-commit hook would make
// checkpoint time scale with subagent fan-out instead of with the commit
// itself. Subagent/workflow turns are still indexed into turns_ft in this
// call — BM25 recall, grouping, and drill-down see them immediately — only
// their nomic vectors are deferred to the next full 'rekal index' or
// 'rekal sync' rebuild, which embeds everything. Recall already discounts
// subagent-session scores relative to trunk turns (docs/agent-metadata.md),
// so a temporarily BM25-only subagent hit is a small, self-correcting gap.
func updateIndexIncremental(gitRoot string, sessionIDs, embeddableSessionIDs []string, checkpointID string, w io.Writer) error {
	indexPath := db.IndexPath(gitRoot)
	if _, err := os.Stat(indexPath); err != nil {
		// No index DB yet — skip incremental update. Next 'rekal index' or 'rekal sync' will build it.
		return nil
	}

	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return err
	}
	defer indexDB.Close()

	// The index may predate optional harness-metadata columns; migrations
	// are additive and idempotent.
	if err := db.MigrateIndexSchema(indexDB); err != nil {
		return fmt.Errorf("migrate index db: %w", err)
	}

	// Populate index tables for new sessions.
	if err := db.PopulateIndexIncremental(indexDB, gitRoot, sessionIDs, checkpointID); err != nil {
		return fmt.Errorf("populate index: %w", err)
	}

	// Knowledge layer: the commit that fired this hook may have touched prose
	// files — refresh chunks for changed blobs now so the layer is fresh even
	// for consumers that don't go through recall's own refresh (raw
	// `rekal query --index`). Watermark-gated and blob-diffed: cost is
	// proportional to the commit's prose delta, usually zero or a few files.
	// Best-effort — the hook must never fail a commit, and recall re-runs the
	// same refresh if this one is skipped.
	if err := db.LoadFTSExtension(indexDB); err != nil {
		fmt.Fprintf(w, "rekal: warning: knowledge refresh skipped: %v\n", err)
	} else if err := refreshKnowledge(nil, indexDB, gitRoot); err != nil {
		fmt.Fprintf(w, "rekal: warning: knowledge refresh failed: %v\n", err)
	} else if err := embedKnowledgeChunks(nil, indexDB, gitRoot, knowledgeEmbedBudget, nil); err != nil {
		// Budgeted: a giant prose import converges over the next few
		// commits; un-embedded chunks stay keyword-findable meanwhile.
		fmt.Fprintf(w, "rekal: warning: knowledge embeddings skipped: %v\n", err)
	}

	// Nomic embeddings for trunk sessions only (non-fatal).
	if len(embeddableSessionIDs) == 0 {
		return nil
	}
	sessionContent, err := db.QuerySessionContentByIDs(indexDB, embeddableSessionIDs)
	if err != nil || len(sessionContent) == 0 {
		return err
	}

	// New trunk sessions only — small set; embed fully (budget 0). Large
	// backfills run via 'rekal embed' after index/sync.
	if _, err := buildSemanticEmbeddings(indexDB, sessionContent, w, gitRoot, 0, nil); err != nil {
		fmt.Fprintf(w, "rekal: warning: nomic embeddings skipped: %v\n", err)
	}

	return nil
}
