package db

import "database/sql"

// InitDataSchema creates the data DB tables if they do not exist.
// Data DB is the source of truth — append-only, never rebuilt.
func InitDataSchema(d *sql.DB) error {
	_, err := d.Exec(dataDDL)
	return err
}

// InitIndexSchema creates the index DB tables if they do not exist.
// Index DB is derived — can be dropped and rebuilt from data DB.
func InitIndexSchema(d *sql.DB) error {
	if indexDDL == "" {
		return nil
	}
	_, err := d.Exec(indexDDL)
	return err
}

const dataDDL = `
CREATE TABLE IF NOT EXISTS sessions (
	id                VARCHAR PRIMARY KEY,
	parent_session_id VARCHAR,
	session_hash      VARCHAR NOT NULL,
	captured_at       TIMESTAMP NOT NULL,
	actor_type        VARCHAR NOT NULL DEFAULT 'human',
	agent_id          VARCHAR,
	user_email        VARCHAR,
	branch            VARCHAR
);

CREATE TABLE IF NOT EXISTS turns (
	id              VARCHAR PRIMARY KEY,
	session_id      VARCHAR NOT NULL REFERENCES sessions(id),
	turn_index      INTEGER NOT NULL,
	role            VARCHAR NOT NULL,
	content         VARCHAR NOT NULL,
	ts              TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tool_calls (
	id              VARCHAR PRIMARY KEY,
	session_id      VARCHAR NOT NULL REFERENCES sessions(id),
	call_order      INTEGER NOT NULL,
	tool            VARCHAR NOT NULL,
	path            VARCHAR,
	cmd_prefix      VARCHAR
);

CREATE TABLE IF NOT EXISTS checkpoints (
	id              VARCHAR PRIMARY KEY,
	git_sha         VARCHAR NOT NULL,
	git_branch      VARCHAR NOT NULL,
	user_email      VARCHAR NOT NULL,
	ts              TIMESTAMP NOT NULL,
	actor_type      VARCHAR NOT NULL DEFAULT 'human',
	agent_id        VARCHAR
);

CREATE TABLE IF NOT EXISTS files_touched (
	id              VARCHAR PRIMARY KEY,
	checkpoint_id   VARCHAR NOT NULL REFERENCES checkpoints(id),
	file_path       VARCHAR NOT NULL,
	change_type     VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS checkpoint_sessions (
	checkpoint_id   VARCHAR NOT NULL REFERENCES checkpoints(id),
	session_id      VARCHAR NOT NULL REFERENCES sessions(id),
	PRIMARY KEY (checkpoint_id, session_id)
);
`

// Index DDL is placeholder — will be defined when index is implemented.
const indexDDL = ``
