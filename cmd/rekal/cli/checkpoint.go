package cli

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/graph"
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
Run manually to capture a session without committing.

Capture is local. Nothing reaches the team until 'rekal push', and push only
shares checkpoints whose work merged. A capture during a rebase is skipped —
git replays every commit, and linking the live conversation to each one would
claim work it never did.

Re-capturing the same conversation appends the new turns to the session that
already exists rather than storing a second copy of it.`,
		Example: `  # What the post-commit hook runs for you
  rekal checkpoint
    rekal: 1 session(s) captured

  # Nothing to capture — silent, exits 0
  rekal checkpoint

  # See what it recorded
  rekal log -n 1
  rekal query -q "SELECT count(*) FROM turns"`,
		Args: rejectExtraArgs,
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
	// A rebase replays commits, and git fires post-commit for every one. Those
	// are not new reasoning — the work already happened and was already
	// captured under its original SHA. Capturing again costs a full pass per
	// replayed commit and, when the agent's own transcript is growing while it
	// rebases, links the live session to commits it never produced. Skip; the
	// next real commit sweeps up everything, since capture reads the whole
	// transcript rather than a delta.
	if gitx.RebaseInProgress(gitRoot) {
		return nil
	}
	return doCheckpointNow(gitRoot, w)
}

func doCheckpointNow(gitRoot string, w io.Writer) error {
	// Open data DB. Closed explicitly before index work (below) so
	// PopulateIndexIncremental's migrate/ATTACH does not open a second
	// connection to the same DuckDB file — a known go-duckdb SIGSEGV hazard.
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data DB: %w", err)
	}
	dataOpen := true
	defer func() {
		if dataOpen {
			_ = dataDB.Close()
		}
	}()

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

			// Resolve to an existing session when this transcript was captured
			// before and has only grown since — a live conversation has
			// different content at every commit, so keying on content stored
			// the whole transcript again each time. The mapping lives in
			// checkpoint_state (local-only, never wired); sessions.session_hash
			// keeps its content-hash meaning, which cross-repo local import
			// depends on. A transcript that was rewritten rather than extended
			// fails the prefix check below and is stored as a new session, so a
			// mismatch can never corrupt the earlier record.
			appendToID := ""
			startTurn, startToolCall := 0, 0
			if existingID, qErr := db.CheckpointStateSessionID(dataDB, cacheKey); qErr == nil && existingID != "" {
				stored, sErr := db.SessionTurnContents(dataDB, existingID)
				if sErr != nil {
					return sErr
				}
				if turnsExtend(stored, payload.Turns) {
					nTurns, nTools, eErr := db.SessionExtent(dataDB, existingID)
					if eErr != nil {
						return eErr
					}
					appendToID, startTurn, startToolCall = existingID, nTurns, nTools
				}
			}

			// Already fully captured — refresh the file cache and move on.
			if appendToID != "" && startTurn >= len(payload.Turns) && startToolCall >= len(payload.ToolCalls) {
				if ref.Path != "" {
					if info, statErr := os.Stat(ref.Path); statErr == nil {
						_ = db.UpsertCheckpointState(dataDB, cacheKey, info.Size(), hash)
					}
				} else {
					_ = db.UpsertCheckpointState(dataDB, cacheKey, 0, hash)
				}
				continue
			}

			sessionID := appendToID
			if sessionID == "" {
				sessionID = newID()
			}
			capturedAt := time.Now().UTC()

			// Resolve the parent session for subagent transcripts: prefer a
			// trunk inserted in this run, then fall back to looking up the
			// trunk file's content hash from a previous capture.
			parentSessionID := ""
			if payload.ParentSessionPath != "" {
				parentSessionID = trunkSessionIDs[payload.ParentSessionPath]
				if parentSessionID == "" {
					// The trunk keeps growing, so its content hash changes
					// between commits. Prefer the checkpoint_state mapping,
					// which is stable per transcript, and fall back to the
					// content hash for trunks captured before the mapping.
					if id, qErr := db.CheckpointStateSessionID(dataDB, payload.ParentSessionPath); qErr == nil && id != "" {
						parentSessionID = id
					} else if trunkData, rdErr := os.ReadFile(payload.ParentSessionPath); rdErr == nil && len(trunkData) > 0 {
						if id, qErr := db.QuerySessionIDByHash(dataDB, sha256Hex(trunkData)); qErr == nil {
							parentSessionID = id
						}
					}
				}
			}

			// Insert the session row only for a conversation not already stored;
			// an append reuses the existing row untouched (append-only holds —
			// turns are added, nothing is rewritten).
			if appendToID == "" {
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
			}
			if payload.ParentSessionPath == "" && ref.Path != "" {
				trunkSessionIDs[ref.Path] = sessionID
			}
			if parentSessionID == "" {
				trunkOnlySessionIDs = append(trunkOnlySessionIDs, sessionID)
			}

			// Insert turns into DuckDB. On an append, start past what is already
			// stored; turn_index keeps counting from there, so the sequence
			// stays contiguous across commits.
			for i := startTurn; i < len(payload.Turns); i++ {
				t := payload.Turns[i]
				ts := ""
				if !t.Timestamp.IsZero() {
					ts = t.Timestamp.UTC().Format(time.RFC3339)
				}
				if err := db.InsertTurn(dataDB, newID(), sessionID, i, t.Role, t.Content, ts); err != nil {
					return fmt.Errorf("insert turn: %w", err)
				}
			}

			// Insert tool calls into DuckDB.
			for i := startToolCall; i < len(payload.ToolCalls); i++ {
				tc := payload.ToolCalls[i]
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

			// Update checkpoint state cache, recording which session this
			// transcript produced so the next capture of it appends rather than
			// storing the whole conversation again.
			if ref.Path != "" {
				info, _ := os.Stat(ref.Path)
				if info != nil {
					_ = db.UpsertCheckpointState(dataDB, cacheKey, info.Size(), hash, sessionID)
				}
			} else {
				_ = db.UpsertCheckpointState(dataDB, cacheKey, 0, hash, sessionID)
			}

			sessionIDs = append(sessionIDs, sessionID)
			inserted++
		}
	}

	// Drain the L1 recall-graph spool into data.db.recall_edges while we still
	// hold the write handle — this is the one place the hot recall path's
	// lock-free appends become the permanent record. Runs even when no new
	// session was captured (an agent may recall/drill inside an
	// already-checkpointed session), so the graph never stalls.
	drained := drainRecallSpool(gitRoot, dataDB, w)

	if inserted == 0 {
		// Nothing new to checkpoint. If we drained edges, the derived
		// session_reach aggregate still needs refreshing — release the writer
		// first (a second live data.db handle can crash under CGO), then refresh.
		if drained > 0 {
			_ = dataDB.Close()
			dataOpen = false
			if err := db.RefreshSessionReach(gitRoot); err != nil {
				fmt.Fprintf(w, "rekal: warning: recall-graph reach refresh failed: %v\n", err)
			}
		}
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

	// (The recall-graph spool was already drained above, before the
	// inserted==0 gate; the incremental index update below refreshes
	// session_reach from data.db.recall_edges.)

	// Release the write handle before index population re-opens/ATTACHes
	// data.db. Two live connections to one DuckDB file in-process can
	// SIGSEGV in CGO (observed in export QueryTurns after checkpoint).
	if err := dataDB.Close(); err != nil {
		return fmt.Errorf("close data DB: %w", err)
	}
	dataOpen = false

	// Incrementally update the index for newly captured sessions.
	if err := updateIndexIncremental(gitRoot, sessionIDs, trunkOnlySessionIDs, checkpointID, w); err != nil {
		// Non-fatal — index can be rebuilt later with 'rekal index'.
		fmt.Fprintf(w, "rekal: warning: incremental index update failed: %v\n", err)
	}

	fmt.Fprintf(w, "rekal: %d session(s) captured\n", inserted)
	return nil
}

// turnsExtend reports whether parsed is the stored turn sequence plus zero or
// more turns appended — the shape a live transcript has when it is checkpointed
// again at a later commit.
//
// This is the guard on the append path. A transcript that was truncated,
// rewritten, or replaced by a different conversation at the same path fails
// here, and capture falls back to storing a new session, so an append can never
// splice unrelated turns onto an existing record.
func turnsExtend(stored []string, parsed []session.Turn) bool {
	if len(parsed) < len(stored) {
		return false
	}
	for i, s := range stored {
		if parsed[i].Content != s {
			return false
		}
	}
	return true
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// drainRecallSpool moves spooled recall-graph edges into data.db.recall_edges,
// the permanent append-only record. Called at checkpoint with the data.db write
// handle held. Returns how many edges landed (0 if none) so the caller can
// decide whether the derived session_reach aggregate needs refreshing.
// Best-effort — the recall graph is an accreting convenience, so a failure here
// warns and leaves the spool for the next checkpoint rather than failing the
// commit.
func drainRecallSpool(gitRoot string, dataDB *sql.DB, w io.Writer) int {
	edges, err := graph.Drain(gitRoot)
	if err != nil {
		fmt.Fprintf(w, "rekal: warning: recall-graph drain failed: %v\n", err)
		// fall through — edges read before the error still land below
	}
	if len(edges) == 0 {
		return 0
	}
	newID := ids.NewULIDFunc()
	rows := make([]db.RecallEdge, 0, len(edges))
	for _, e := range edges {
		rows = append(rows, db.RecallEdge{
			ID:     newID(),
			TS:     e.TS,
			Kind:   e.Kind,
			Query:  e.Query,
			Target: e.Target,
		})
	}
	if err := db.InsertRecallEdges(dataDB, rows); err != nil {
		fmt.Fprintf(w, "rekal: warning: recall-graph insert failed: %v\n", err)
		return 0
	}
	return len(rows)
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
