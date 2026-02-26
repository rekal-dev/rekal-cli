package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// OpenData opens (or creates) the data DB at <gitRoot>/.rekal/data.db.
func OpenData(gitRoot string) (*sql.DB, error) {
	path := filepath.Join(gitRoot, ".rekal", "data.db")
	return open(path)
}

// OpenIndex opens (or creates) the index DB at <gitRoot>/.rekal/index.db.
func OpenIndex(gitRoot string) (*sql.DB, error) {
	path := filepath.Join(gitRoot, ".rekal", "index.db")
	return open(path)
}

func open(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database %s: %w", path, err)
	}
	return db, nil
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

// InsertSession inserts a new session row into the data DB.
func InsertSession(d *sql.DB, id, parentSessionID, hash, actorType, agentID, userEmail, branch, capturedAt string) error {
	_, err := d.Exec(
		`INSERT INTO sessions (id, parent_session_id, session_hash, captured_at, actor_type, agent_id, user_email, branch)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, nullIfEmpty(parentSessionID), hash, capturedAt, actorType, agentID, userEmail, branch,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// nullIfEmpty returns nil if s is empty, otherwise s.
// Used to store NULL in VARCHAR columns instead of empty strings.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// InsertTurn inserts a turn row into the data DB.
func InsertTurn(d *sql.DB, id, sessionID string, turnIndex int, role, content, ts string) error {
	_, err := d.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content, ts)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, sessionID, turnIndex, role, content, nullIfEmpty(ts),
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
func UpsertCheckpointState(d *sql.DB, filePath string, byteSize int64, fileHash string) error {
	_, err := d.Exec(
		`INSERT INTO checkpoint_state (file_path, byte_size, file_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (file_path) DO UPDATE SET byte_size = $2, file_hash = $3`,
		filePath, byteSize, fileHash,
	)
	if err != nil {
		return fmt.Errorf("upsert checkpoint_state: %w", err)
	}
	return nil
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
}

// QueryUnexportedCheckpoints returns checkpoints where exported = FALSE, ordered by ts.
func QueryUnexportedCheckpoints(d *sql.DB) ([]CheckpointRow, error) {
	rows, err := d.Query(
		`SELECT id, git_sha, git_branch, user_email, ts, actor_type, COALESCE(agent_id, '')
		 FROM checkpoints WHERE exported = FALSE ORDER BY ts`,
	)
	if err != nil {
		return nil, fmt.Errorf("query unexported checkpoints: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []CheckpointRow
	for rows.Next() {
		var r CheckpointRow
		if err := rows.Scan(&r.ID, &r.GitSHA, &r.GitBranch, &r.Email, &r.Ts, &r.ActorType, &r.AgentID); err != nil {
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
		`SELECT id, session_hash, captured_at, actor_type, COALESCE(agent_id, ''), COALESCE(user_email, ''), COALESCE(branch, '')
		 FROM sessions WHERE id = $1`, id,
	).Scan(&r.ID, &r.Hash, &r.CapturedAt, &r.ActorType, &r.AgentID, &r.Email, &r.Branch)
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}
	return r, nil
}

// TurnPageOptions controls pagination and filtering for QueryTurnsPage.
type TurnPageOptions struct {
	Offset int
	Limit  int
	Role   string // "" = all, "human", "assistant"
}

// QueryTurnsPage returns a page of turns for a session with optional role filtering.
// It returns the matching turns, the total count (respecting the role filter), and any error.
func QueryTurnsPage(d *sql.DB, sessionID string, opts TurnPageOptions) ([]TurnRow, int, error) {
	// Build WHERE clause.
	where := "session_id = $1"
	args := []interface{}{sessionID}
	if opts.Role != "" {
		where += " AND role = $2"
		args = append(args, opts.Role)
	}

	// Count total matching turns.
	var total int
	if err := d.QueryRow("SELECT COUNT(*) FROM turns WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count turns: %w", err)
	}

	// Build paginated query.
	q := "SELECT turn_index, role, content, COALESCE(CAST(ts AS VARCHAR), '') FROM turns WHERE " + where + " ORDER BY turn_index"
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
	rows, err := d.Query(
		`SELECT call_order, tool, COALESCE(path, ''), COALESCE(cmd_prefix, '')
		 FROM tool_calls WHERE session_id = $1 ORDER BY call_order`, sessionID,
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
