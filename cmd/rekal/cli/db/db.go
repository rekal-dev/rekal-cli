package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"

	_ "github.com/marcboeker/go-duckdb"
)

// DuckDB is single-writer per file: while a background `rekal embed` bite holds
// the index write lock, any other rekal process opening the same file gets a
// lock-conflict error. Rather than hard-fail (a user running recall right after
// `init`/`sync` would just see an error), the open path waits out a bite. Embed
// holds the lock only for a short, bounded bite and yields between bites (see
// embed_cmd.go), so a reader polling here wins a yield window within one bite.
const (
	// Budget covers the worst realistic hold: the first background-embed bite
	// after a cold start, where the nomic daemon loads the model (~15s) while
	// holding the lock. Steady-state bites are far shorter (see embed_cmd.go),
	// so a reader normally waits well under a second. A genuinely wedged writer
	// still surfaces the clear error after this budget rather than hanging.
	openLockRetryBudget = 30 * time.Second
	openLockRetryMin    = 50 * time.Millisecond
	openLockRetryMax    = 200 * time.Millisecond
)

// StoreDir returns the .rekal store directory for gitRoot, resolved to the
// repository's main worktree so every linked worktree shares one store. For a
// non-worktree repo (or the main worktree itself) this is just <gitRoot>/.rekal.
func StoreDir(gitRoot string) string {
	return filepath.Join(gitx.MainWorktreeRoot(gitRoot), ".rekal")
}

// OpenData opens (or creates) the data DB at <store>/data.db.
func OpenData(gitRoot string) (*sql.DB, error) {
	return open(filepath.Join(StoreDir(gitRoot), "data.db"))
}

// OpenIndex opens (or creates) the index DB at <store>/index.db.
func OpenIndex(gitRoot string) (*sql.DB, error) {
	return OpenIndexAt(IndexPath(gitRoot))
}

// IndexPath returns the path to the index DB for gitRoot (in the shared store).
func IndexPath(gitRoot string) string {
	return filepath.Join(StoreDir(gitRoot), "index.db")
}

// OpenIndexAt opens (or creates) an index DB at an arbitrary path. Used by
// `rekal index`'s rebuild-into-a-temp-file-then-rename path (see
// index_cmd.go) so a rebuild that's interrupted midway never leaves the
// live index.db half-dropped/half-repopulated.
func OpenIndexAt(path string) (*sql.DB, error) {
	return open(path)
}

// OpenDataReadOnly opens the data DB with DuckDB's read-only access mode.
//
// data.db is the append-only ledger, so a read path must not be able to write
// to it at all — the guarantee belongs in the handle, not in whatever parsing
// the caller does first. `rekal query` hands user (and agent) SQL straight to
// the driver, and the driver executes every statement in the string it is
// given: a statement guard that inspects only the leading keyword lets
// "SELECT 1; DELETE FROM turns" through. Read-only makes that structural
// instead of a rule to keep getting right.
func OpenDataReadOnly(gitRoot string) (*sql.DB, error) {
	return openReadOnly(DataPath(gitRoot))
}

// OpenIndexReadOnly opens the index DB read-only. The index is derived and
// rebuildable, but the same reasoning applies: a query path has no business
// holding a writable handle.
func OpenIndexReadOnly(gitRoot string) (*sql.DB, error) {
	return openReadOnly(IndexPath(gitRoot))
}

// DataPath returns the path to the data DB for gitRoot (in the shared store).
func DataPath(gitRoot string) string {
	return filepath.Join(StoreDir(gitRoot), "data.db")
}

// openReadOnly opens an existing DB file in DuckDB's read-only access mode.
// DuckDB cannot create a database read-only, so a missing file is reported as
// such rather than surfacing as a driver error.
func openReadOnly(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	return openDSN(path+"?access_mode=read_only", path)
}

func open(path string) (*sql.DB, error) {
	return openDSN(path, path)
}

// openDSN opens dsn, reporting failures against the plain file path so the
// user never sees the connection-string decoration in an error message.
func openDSN(dsn, path string) (*sql.DB, error) {
	deadline := time.Now().Add(openLockRetryBudget)
	wait := openLockRetryMin
	for {
		// The single-writer lock conflict can surface either at sql.Open (the
		// go-duckdb connector opens the file eagerly) or at Ping — handle both
		// through one retry path.
		db, err := sql.Open("duckdb", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				// DuckDB + go-duckdb are not safe with database/sql's default
				// multi-connection pool: concurrent pooled conns race on WAL
				// auto-checkpoint and can FATAL/SIGSEGV inside CGO (see
				// duckdb/duckdb-go#127). Cap the pool to one connection.
				db.SetMaxOpenConns(1)
				db.SetMaxIdleConns(1)
				return db, nil
			}
			db.Close()
		}
		// Only a lock conflict is worth waiting out, and only until the budget
		// runs out; anything else (corruption, disk full) fails immediately.
		if !isLockConflict(err) || !time.Now().Before(deadline) {
			return nil, wrapOpenError(path, err)
		}
		time.Sleep(wait)
		if wait *= 2; wait > openLockRetryMax {
			wait = openLockRetryMax
		}
	}
}

// isLockConflict reports whether err is DuckDB's single-writer lock conflict —
// another rekal process already has the file open read-write.
func isLockConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Could not set lock on file") ||
		strings.Contains(msg, "Conflicting lock is held")
}

// wrapOpenError translates a DuckDB open/ping failure into a clear message.
// DuckDB is single-writer per file: a second rekal process (e.g. a
// post-commit checkpoint hook firing while a manual `rekal index` runs)
// opening the same data.db/index.db gets a raw driver error whose real cause
// — "another process already has this file open" — is buried in DuckDB's
// own wording. Detect that specific case and say so plainly; anything else
// passes through unchanged rather than risk misinterpreting a genuinely
// different failure (corrupt file, disk full, ...) as a lock conflict.
func wrapOpenError(path string, err error) error {
	if isLockConflict(err) {
		return fmt.Errorf("rekal: another rekal process is already using %s — wait for it to finish and try again: %w", path, err)
	}
	if isUnreadableStorage(err) {
		return fmt.Errorf("rekal: %s is unreadable (DuckDB storage corruption or incompatible binary format) — run `rekal clean && rekal init` to rebuild the local store from your rekal orphan branch: %w", path, err)
	}
	return fmt.Errorf("open database %s: %w", path, err)
}

// isUnreadableStorage reports DuckDB binary deserialize failures — typically
// a truncated/corrupted file (e.g. killed mid-write, multi-connection WAL
// races before MaxOpenConns(1), cloud-synced store) or a file written by a
// newer DuckDB than this binary embeds.
func isUnreadableStorage(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Failed to deserialize") ||
		strings.Contains(msg, "Serialization Error") ||
		strings.Contains(msg, "field id mismatch")
}

// SessionExistsByHash reports whether a session with the given content hash
// already exists in the data DB. Used for deduplication.
func SessionExistsByHash(d *sql.DB, hash string) (bool, error) {
	var count int
	err := d.QueryRow("SELECT count(*) FROM sessions WHERE session_hash = $1", hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check session hash: %w", err)
	}
	return count > 0, nil
}

// QueryAllSessionHashes returns the set of content hashes of every session in
// the data DB. The cross-repo local import uses it to skip sessions that this
// repo already captured (dedup by content, not ID).
func QueryAllSessionHashes(d *sql.DB) (map[string]bool, error) {
	rows, err := d.Query("SELECT session_hash FROM sessions")
	if err != nil {
		return nil, fmt.Errorf("query session hashes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	hashes := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("scan session hash: %w", err)
		}
		hashes[h] = true
	}
	return hashes, rows.Err()
}

// QuerySessionIDByHash returns the ID of the most recently captured session
// with the given content hash. Used to link subagent sessions to a trunk
// session captured in an earlier checkpoint run.
func QuerySessionIDByHash(d *sql.DB, hash string) (string, error) {
	var id string
	err := d.QueryRow(
		`SELECT id FROM sessions WHERE session_hash = $1 ORDER BY captured_at DESC LIMIT 1`, hash,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("query session by hash: %w", err)
	}
	return id, nil
}

// SessionExtent reports how much of a session is already stored: its turn count
// and tool-call count. Used by capture to append only what is new when the same
// transcript is checkpointed again at a later commit.
func SessionExtent(d *sql.DB, sessionID string) (turns, toolCalls int, err error) {
	if err = d.QueryRow(
		`SELECT count(*) FROM turns WHERE session_id = $1`, sessionID,
	).Scan(&turns); err != nil {
		return 0, 0, fmt.Errorf("count turns: %w", err)
	}
	if err = d.QueryRow(
		`SELECT count(*) FROM tool_calls WHERE session_id = $1`, sessionID,
	).Scan(&toolCalls); err != nil {
		return 0, 0, fmt.Errorf("count tool_calls: %w", err)
	}
	return turns, toolCalls, nil
}

// SessionTurnContents returns the stored turn contents for a session, ordered by
// turn_index. Capture compares these against a freshly parsed transcript to
// confirm it is a strict extension before appending — a transcript that was
// rewritten rather than extended must not be merged into the existing session.
func SessionTurnContents(d *sql.DB, sessionID string) ([]string, error) {
	rows, err := d.Query(
		`SELECT content FROM turns WHERE session_id = $1 ORDER BY turn_index`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turn contents: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("scan turn content: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SessionMetaFields bundles optional harness metadata for InsertSessionMeta:
// team/workflow (Claude Code teammates runs and dynamic workflows) and
// agent-type/description/spawn-depth (Task subagent meta.json sidecars —
// see claude.go's subagentMeta doc for the real observed shape). Every field
// is optional; the zero value means "not applicable for this harness or
// session" and is stored as NULL, not as an empty string/zero.
type SessionMetaFields struct {
	TeamName     string
	WorkflowName string
	AgentType    string
	Description  string
	SpawnDepth   int // 0 means "not set" — real subagent depths start at 1
}

// InsertSession inserts a new session row into the data DB.
func InsertSession(d *sql.DB, id, parentSessionID, hash, actorType, agentID, userEmail, branch, capturedAt, source string) error {
	return InsertSessionMeta(d, id, parentSessionID, hash, actorType, agentID, userEmail, branch, capturedAt, source, SessionMetaFields{})
}

// InsertSessionMeta is InsertSession with optional harness metadata for
// sessions captured from Claude Code teammates runs, dynamic workflows, or
// Task subagents.
func InsertSessionMeta(d *sql.DB, id, parentSessionID, hash, actorType, agentID, userEmail, branch, capturedAt, source string, meta SessionMetaFields) error {
	if source == "" {
		source = "claude"
	}
	_, err := d.Exec(
		`INSERT INTO sessions (id, parent_session_id, session_hash, captured_at, actor_type, agent_id, user_email, branch, source, team_name, workflow_name, agent_type, description, spawn_depth)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		id, NullIfEmpty(parentSessionID), hash, capturedAt, actorType, agentID, userEmail, branch, source,
		NullIfEmpty(meta.TeamName), NullIfEmpty(meta.WorkflowName),
		NullIfEmpty(meta.AgentType), NullIfEmpty(meta.Description), NullIfZero(meta.SpawnDepth),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// NullIfEmpty returns nil if s is empty, otherwise s. Used to store NULL in
// VARCHAR columns instead of empty strings, from both this package's insert
// helpers and callers that build raw INSERTs against these tables.
func NullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// NullIfZero returns nil if n is zero, otherwise n. Used to store NULL in
// INTEGER columns instead of a misleading 0.
func NullIfZero(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

// InsertTurn inserts a turn row into the data DB.
func InsertTurn(d *sql.DB, id, sessionID string, turnIndex int, role, content, ts string) error {
	_, err := d.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content, ts)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, sessionID, turnIndex, role, content, NullIfEmpty(ts),
	)
	if err != nil {
		return fmt.Errorf("insert turn: %w", err)
	}
	return nil
}

// InsertToolCall inserts a tool_call row into the data DB.
func InsertToolCall(d *sql.DB, id, sessionID string, callOrder int, tool, path, cmdPrefix string) error {
	_, err := d.Exec(
		`INSERT INTO tool_calls (id, session_id, call_order, tool, path, cmd_prefix)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, sessionID, callOrder, tool, path, cmdPrefix,
	)
	if err != nil {
		return fmt.Errorf("insert tool_call: %w", err)
	}
	return nil
}

// InsertCheckpoint inserts a new checkpoint row into the data DB.
func InsertCheckpoint(d *sql.DB, id, gitSHA, branch, email, ts, actorType, agentID string) error {
	_, err := d.Exec(
		`INSERT INTO checkpoints (id, git_sha, git_branch, user_email, ts, actor_type, agent_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, gitSHA, branch, email, ts, actorType, agentID,
	)
	if err != nil {
		return fmt.Errorf("insert checkpoint: %w", err)
	}
	return nil
}

// InsertFileTouched inserts a file_touched row.
func InsertFileTouched(d *sql.DB, id, checkpointID, filePath, changeType string) error {
	_, err := d.Exec(
		`INSERT INTO files_touched (id, checkpoint_id, file_path, change_type)
		 VALUES ($1, $2, $3, $4)`,
		id, checkpointID, filePath, changeType,
	)
	if err != nil {
		return fmt.Errorf("insert file_touched: %w", err)
	}
	return nil
}

// InsertCheckpointSession inserts a checkpoint_sessions junction row.
func InsertCheckpointSession(d *sql.DB, checkpointID, sessionID string) error {
	_, err := d.Exec(
		`INSERT INTO checkpoint_sessions (checkpoint_id, session_id)
		 VALUES ($1, $2)`,
		checkpointID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("insert checkpoint_session: %w", err)
	}
	return nil
}

// GetCheckpointState returns the cached state for a session file path.
// Returns found=false if no entry exists.
func GetCheckpointState(d *sql.DB, filePath string) (byteSize int64, fileHash string, found bool, err error) {
	err = d.QueryRow(
		"SELECT byte_size, file_hash FROM checkpoint_state WHERE file_path = $1",
		filePath,
	).Scan(&byteSize, &fileHash)
	if err == sql.ErrNoRows {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("get checkpoint_state: %w", err)
	}
	return byteSize, fileHash, true, nil
}

// UpsertCheckpointState inserts or updates the cached state for a session file.
// A non-empty sessionID records which session that transcript produced; passing
// "" leaves any previously recorded mapping intact.
func UpsertCheckpointState(d *sql.DB, filePath string, byteSize int64, fileHash string, sessionID ...string) error {
	sid := ""
	if len(sessionID) > 0 {
		sid = sessionID[0]
	}
	var err error
	if sid == "" {
		_, err = d.Exec(
			`INSERT INTO checkpoint_state (file_path, byte_size, file_hash)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (file_path) DO UPDATE SET byte_size = $2, file_hash = $3`,
			filePath, byteSize, fileHash,
		)
	} else {
		_, err = d.Exec(
			`INSERT INTO checkpoint_state (file_path, byte_size, file_hash, session_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (file_path) DO UPDATE SET byte_size = $2, file_hash = $3, session_id = $4`,
			filePath, byteSize, fileHash, sid,
		)
	}
	if err != nil {
		return fmt.Errorf("upsert checkpoint_state: %w", err)
	}
	return nil
}

// CheckpointStateSessionID returns the session a previously captured transcript
// produced, or "" when the path is unknown or predates the mapping (data.db
// written by an older rekal). An empty result means capture stores a new
// session, which is the pre-existing behaviour.
func CheckpointStateSessionID(d *sql.DB, filePath string) (string, error) {
	var sid sql.NullString
	err := d.QueryRow(
		"SELECT session_id FROM checkpoint_state WHERE file_path = $1", filePath,
	).Scan(&sid)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read checkpoint_state session_id: %w", err)
	}
	return sid.String, nil
}

// CheckpointRow represents a row from the checkpoints table.
type CheckpointRow struct {
	ID        string
	GitSHA    string
	GitBranch string
	Email     string
	Ts        string
	ActorType string
	AgentID   string
	// Exported records whether this checkpoint already reached the wire. A
	// repair (push --re-export) uses it as standing proof of the merged-only
	// gate: the checkpoint passed that gate once, and its commit may since
	// have been rebased or squashed away, leaving nothing to re-prove with.
	Exported bool
}

// QueryUnexportedCheckpoints returns checkpoints where exported = FALSE, ordered by ts.
func QueryUnexportedCheckpoints(d *sql.DB) ([]CheckpointRow, error) {
	return queryCheckpoints(d, "WHERE exported = FALSE")
}

// QueryAllCheckpoints returns every checkpoint ordered by ts, regardless of
// exported state. Used by `rekal push --re-export` to regenerate the orphan
// branch's wire data from scratch.
func QueryAllCheckpoints(d *sql.DB) ([]CheckpointRow, error) {
	return queryCheckpoints(d, "")
}

func queryCheckpoints(d *sql.DB, where string) ([]CheckpointRow, error) {
	rows, err := d.Query(
		`SELECT id, git_sha, git_branch, user_email, ts, actor_type, COALESCE(agent_id, ''), COALESCE(exported, FALSE)
		 FROM checkpoints ` + where + ` ORDER BY ts`,
	)
	if err != nil {
		return nil, fmt.Errorf("query checkpoints: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []CheckpointRow
	for rows.Next() {
		var r CheckpointRow
		if err := rows.Scan(&r.ID, &r.GitSHA, &r.GitBranch, &r.Email, &r.Ts, &r.ActorType, &r.AgentID, &r.Exported); err != nil {
			return nil, fmt.Errorf("scan checkpoint: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// MarkCheckpointsExported sets exported = TRUE for the given checkpoint IDs.
func MarkCheckpointsExported(d *sql.DB, ids []string) error {
	for _, id := range ids {
		if _, err := d.Exec("UPDATE checkpoints SET exported = TRUE WHERE id = $1", id); err != nil {
			return fmt.Errorf("mark checkpoint exported: %w", err)
		}
	}
	return nil
}

// QuerySessionsByCheckpoint returns session IDs linked to a checkpoint.
func QuerySessionsByCheckpoint(d *sql.DB, checkpointID string) ([]string, error) {
	rows, err := d.Query(
		"SELECT session_id FROM checkpoint_sessions WHERE checkpoint_id = $1",
		checkpointID,
	)
	if err != nil {
		return nil, fmt.Errorf("query checkpoint sessions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SessionRow represents a session with its turns and tool calls.
type SessionRow struct {
	ID         string
	Hash       string
	CapturedAt string
	ActorType  string
	AgentID    string
	Email      string
	Branch     string

	// Optional harness metadata — empty for agents without the concept.
	TeamName        string
	WorkflowName    string
	ParentSessionID string

	// Optional Task subagent meta.json sidecar fields — empty for sessions
	// that aren't a subagent transcript, or that predate this capture.
	AgentType   string
	Description string
	SpawnDepth  int
}

// TurnRow represents a turn from the turns table.
type TurnRow struct {
	TurnIndex int
	Role      string
	Content   string
	Ts        string
}

// ToolCallRow represents a tool call from the tool_calls table.
type ToolCallRow struct {
	CallOrder int
	Tool      string
	Path      string
	CmdPrefix string
}

// QuerySession returns a session row by ID.
func QuerySession(d *sql.DB, id string) (*SessionRow, error) {
	r := &SessionRow{}
	err := d.QueryRow(
		`SELECT id, session_hash, captured_at, actor_type, COALESCE(agent_id, ''), COALESCE(user_email, ''), COALESCE(branch, ''),
		        COALESCE(team_name, ''), COALESCE(workflow_name, ''), COALESCE(parent_session_id, ''),
		        COALESCE(agent_type, ''), COALESCE(description, ''), COALESCE(spawn_depth, 0)
		 FROM sessions WHERE id = $1`, id,
	).Scan(&r.ID, &r.Hash, &r.CapturedAt, &r.ActorType, &r.AgentID, &r.Email, &r.Branch,
		&r.TeamName, &r.WorkflowName, &r.ParentSessionID,
		&r.AgentType, &r.Description, &r.SpawnDepth)
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}
	return r, nil
}

// QuerySessionFromIndex returns a session's metadata from the index DB
// (session_facets). Teammate sessions pulled via `rekal sync` exist only in
// the index — data.db is owner-only by design — so drill-down falls back to
// this lookup when a session is not in the data DB. The content hash is not
// stored in session_facets, so Hash is empty.
func QuerySessionFromIndex(d *sql.DB, id string) (*SessionRow, error) {
	r := &SessionRow{}
	err := d.QueryRow(
		`SELECT session_id, CAST(captured_at AS VARCHAR), actor_type, COALESCE(agent_id, ''), COALESCE(user_email, ''), COALESCE(git_branch, ''),
		        COALESCE(team_name, ''), COALESCE(workflow_name, ''), COALESCE(parent_session_id, ''),
		        COALESCE(agent_type, ''), COALESCE(description, ''), COALESCE(spawn_depth, 0)
		 FROM session_facets WHERE session_id = $1`, id,
	).Scan(&r.ID, &r.CapturedAt, &r.ActorType, &r.AgentID, &r.Email, &r.Branch,
		&r.TeamName, &r.WorkflowName, &r.ParentSessionID,
		&r.AgentType, &r.Description, &r.SpawnDepth)
	if err != nil {
		return nil, fmt.Errorf("query session from index: %w", err)
	}
	return r, nil
}

// QueryChildSessionIDs returns the IDs of sessions whose parent_session_id
// points at sessionID, ordered by capture time — the subagent/workflow
// transcripts folded under this trunk conversation. Generic across agent
// types: returns empty for sessions with no children.
func QueryChildSessionIDs(d *sql.DB, sessionID string) ([]string, error) {
	rows, err := d.Query(
		`SELECT id FROM sessions WHERE parent_session_id = $1 ORDER BY captured_at`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query child sessions: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	return scanIDs(rows)
}

// QueryChildSessionIDsFromIndex is QueryChildSessionIDs against the index DB
// (session_facets), used for remote/teammate sessions.
func QueryChildSessionIDsFromIndex(d *sql.DB, sessionID string) ([]string, error) {
	rows, err := d.Query(
		`SELECT session_id FROM session_facets WHERE parent_session_id = $1 ORDER BY captured_at`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query child sessions from index: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	return scanIDs(rows)
}

func scanIDs(rows *sql.Rows) ([]string, error) {
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// TurnPageOptions controls pagination and filtering for QueryTurnsPage.
type TurnPageOptions struct {
	Offset int
	Limit  int
	Role   string // "" = all; exact match otherwise: "human", "assistant", "human_steering"
}

// QueryTurnsPage returns a page of turns for a session with optional role filtering.
// It returns the matching turns, the total count (respecting the role filter), and any error.
func QueryTurnsPage(d *sql.DB, sessionID string, opts TurnPageOptions) ([]TurnRow, int, error) {
	return queryTurnsPageFrom(d, "turns", sessionID, opts)
}

// QueryTurnsPageFromIndex is QueryTurnsPage against the index DB (turns_ft),
// used for remote/teammate sessions that are not present in the data DB.
func QueryTurnsPageFromIndex(d *sql.DB, sessionID string, opts TurnPageOptions) ([]TurnRow, int, error) {
	return queryTurnsPageFrom(d, "turns_ft", sessionID, opts)
}

func queryTurnsPageFrom(d *sql.DB, table, sessionID string, opts TurnPageOptions) ([]TurnRow, int, error) {
	// data.db rows written before the "summary" role existed store compaction
	// summaries as "human" (append-only, never rewritten), so roles there are
	// read through the source-scoped reclassification (see
	// SummaryFingerprint). turns_ft rows are written already-reclassified by
	// index population / sync import — read them as stored.
	roleExpr := "role"
	if table == "turns" {
		roleExpr = summaryRoleExprData
	}

	// Build WHERE clause.
	where := "session_id = $1"
	args := []interface{}{sessionID}
	if opts.Role != "" {
		where += " AND " + roleExpr + " = $2"
		args = append(args, opts.Role)
	}

	// Count total matching turns.
	var total int
	if err := d.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count turns: %w", err)
	}

	// Build paginated query.
	q := "SELECT turn_index, " + roleExpr + " AS role, content, COALESCE(CAST(ts AS VARCHAR), '') FROM " + table + " WHERE " + where + " ORDER BY turn_index"
	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query turns page: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []TurnRow
	for rows.Next() {
		var r TurnRow
		if err := rows.Scan(&r.TurnIndex, &r.Role, &r.Content, &r.Ts); err != nil {
			return nil, 0, fmt.Errorf("scan turn: %w", err)
		}
		result = append(result, r)
	}
	return result, total, rows.Err()
}

// QueryTurns returns turns for a session, ordered by turn_index.
func QueryTurns(d *sql.DB, sessionID string) ([]TurnRow, error) {
	rows, err := d.Query(
		`SELECT turn_index, role, content, COALESCE(CAST(ts AS VARCHAR), '')
		 FROM turns WHERE session_id = $1 ORDER BY turn_index`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []TurnRow
	for rows.Next() {
		var r TurnRow
		if err := rows.Scan(&r.TurnIndex, &r.Role, &r.Content, &r.Ts); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryToolCalls returns tool calls for a session, ordered by call_order.
func QueryToolCalls(d *sql.DB, sessionID string) ([]ToolCallRow, error) {
	return queryToolCallsFrom(d, "tool_calls", sessionID)
}

// QueryToolCallsFromIndex is QueryToolCalls against the index DB
// (tool_calls_index), used for remote/teammate sessions.
func QueryToolCallsFromIndex(d *sql.DB, sessionID string) ([]ToolCallRow, error) {
	return queryToolCallsFrom(d, "tool_calls_index", sessionID)
}

func queryToolCallsFrom(d *sql.DB, table, sessionID string) ([]ToolCallRow, error) {
	rows, err := d.Query(
		`SELECT call_order, tool, COALESCE(path, ''), COALESCE(cmd_prefix, '')
		 FROM `+table+` WHERE session_id = $1 ORDER BY call_order`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query tool_calls: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []ToolCallRow
	for rows.Next() {
		var r ToolCallRow
		if err := rows.Scan(&r.CallOrder, &r.Tool, &r.Path, &r.CmdPrefix); err != nil {
			return nil, fmt.Errorf("scan tool_call: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryFilesTouched returns files touched for a checkpoint.
func QueryFilesTouched(d *sql.DB, checkpointID string) ([]struct{ Path, ChangeType string }, error) {
	rows, err := d.Query(
		"SELECT file_path, change_type FROM files_touched WHERE checkpoint_id = $1",
		checkpointID,
	)
	if err != nil {
		return nil, fmt.Errorf("query files_touched: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []struct{ Path, ChangeType string }
	for rows.Next() {
		var r struct{ Path, ChangeType string }
		if err := rows.Scan(&r.Path, &r.ChangeType); err != nil {
			return nil, fmt.Errorf("scan file_touched: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CheckpointExists reports whether a checkpoint with the given ID exists.
func CheckpointExists(d *sql.DB, id string) (bool, error) {
	var count int
	err := d.QueryRow("SELECT count(*) FROM checkpoints WHERE id = $1", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check checkpoint exists: %w", err)
	}
	return count > 0, nil
}

// SessionExistsByID reports whether a session with the given ID exists.
func SessionExistsByID(d *sql.DB, id string) (bool, error) {
	var count int
	err := d.QueryRow("SELECT count(*) FROM sessions WHERE id = $1", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check session id: %w", err)
	}
	return count > 0, nil
}
