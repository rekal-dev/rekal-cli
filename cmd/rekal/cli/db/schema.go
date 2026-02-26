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
	agent_id        VARCHAR,
	exported        BOOLEAN NOT NULL DEFAULT FALSE
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

CREATE TABLE IF NOT EXISTS checkpoint_state (
	file_path   VARCHAR PRIMARY KEY,
	byte_size   BIGINT NOT NULL,
	file_hash   VARCHAR NOT NULL
);
`

// Index DDL defines the derived index tables — rebuilt from data DB.
const indexDDL = `
CREATE TABLE IF NOT EXISTS turns_ft (
	id              VARCHAR PRIMARY KEY,
	session_id      VARCHAR NOT NULL,
	turn_index      INTEGER NOT NULL,
	role            VARCHAR NOT NULL,
	content         VARCHAR NOT NULL,
	ts              VARCHAR
);

CREATE TABLE IF NOT EXISTS tool_calls_index (
	id              VARCHAR PRIMARY KEY,
	session_id      VARCHAR NOT NULL,
	call_order      INTEGER NOT NULL,
	tool            VARCHAR NOT NULL,
	path            VARCHAR,
	cmd_prefix      VARCHAR
);
CREATE INDEX IF NOT EXISTS idx_tci_tool ON tool_calls_index(tool);
CREATE INDEX IF NOT EXISTS idx_tci_path ON tool_calls_index(path);
CREATE INDEX IF NOT EXISTS idx_tci_session ON tool_calls_index(session_id);

CREATE TABLE IF NOT EXISTS files_index (
	checkpoint_id   VARCHAR NOT NULL,
	session_id      VARCHAR NOT NULL,
	file_path       VARCHAR NOT NULL,
	change_type     VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fi_path ON files_index(file_path);
CREATE INDEX IF NOT EXISTS idx_fi_session ON files_index(session_id);

CREATE TABLE IF NOT EXISTS session_facets (
	session_id      VARCHAR PRIMARY KEY,
	user_email      VARCHAR,
	git_branch      VARCHAR,
	actor_type      VARCHAR NOT NULL,
	agent_id        VARCHAR,
	captured_at     TIMESTAMP NOT NULL,
	turn_count      INTEGER NOT NULL DEFAULT 0,
	tool_call_count INTEGER NOT NULL DEFAULT 0,
	file_count      INTEGER NOT NULL DEFAULT 0,
	checkpoint_id   VARCHAR,
	git_sha         VARCHAR
);
CREATE INDEX IF NOT EXISTS idx_sf_email ON session_facets(user_email);
CREATE INDEX IF NOT EXISTS idx_sf_actor ON session_facets(actor_type);
CREATE INDEX IF NOT EXISTS idx_sf_branch ON session_facets(git_branch);
CREATE INDEX IF NOT EXISTS idx_sf_sha ON session_facets(git_sha);

CREATE TABLE IF NOT EXISTS file_cooccurrence (
	file_a          VARCHAR NOT NULL,
	file_b          VARCHAR NOT NULL,
	count           INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (file_a, file_b)
);

CREATE TABLE IF NOT EXISTS session_embeddings (
	session_id      VARCHAR PRIMARY KEY,
	embedding       FLOAT[],
	model           VARCHAR NOT NULL,
	generated_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS index_state (
	key             VARCHAR PRIMARY KEY,
	value           VARCHAR NOT NULL
);
`
