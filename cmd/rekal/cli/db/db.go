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
		id, sessionID, turnIndex, role, content, ts,
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
