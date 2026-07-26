package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CurrentDataSchemaVersion is the highest data.db schema version this rekal
// build understands. Bump it whenever a change to dataDDL/MigrateDataSchema
// is not purely additive-and-ignorable by older builds (see
// SchemaVersionError doc for why this matters).
const CurrentDataSchemaVersion = 1

// CurrentIndexSchemaVersion is the index.db equivalent of
// CurrentDataSchemaVersion.
const CurrentIndexSchemaVersion = 1

// SchemaVersionError means a DB's stamped schema version is newer than this
// rekal build understands — see the forward-compatibility design note in
// docs/db/ (data.db is append-only and shared via git, so a team routinely
// has members on different rekal versions; someone on an older build must
// get a clear error here, not a confusing failure three layers into a query
// against columns/tables it doesn't know about).
type SchemaVersionError struct {
	DBName     string // "data.db" or "index.db"
	DBVersion  int
	MaxVersion int
}

func (e *SchemaVersionError) Error() string {
	return fmt.Sprintf(
		"%s was written by a newer rekal (schema v%d) — this build only understands up to v%d. Upgrade rekal to continue.",
		e.DBName, e.DBVersion, e.MaxVersion,
	)
}

// InitDataSchema creates the data DB tables if they do not exist.
// Data DB is the source of truth — append-only, never rebuilt.
func InitDataSchema(d *sql.DB) error {
	if _, err := d.Exec(dataDDL); err != nil {
		return err
	}
	return MigrateDataSchema(d)
}

// MigrateDataSchema applies forward-only migrations to an existing data DB.
// Safe to call multiple times — each migration checks before applying.
// Migrations must stay additive: data.db is append-only and shared via git,
// so DBs written by older rekal versions must keep opening cleanly.
//
// Before touching anything else, it checks the DB's stamped schema version
// against CurrentDataSchemaVersion and refuses to proceed if the DB is
// newer than this build understands (SchemaVersionError) — see the
// forward-compatibility design note in docs/db/.
func MigrateDataSchema(d *sql.DB) error {
	// schema_meta may not exist yet on a data.db written before this
	// version-stamping was added (every call site here reaches
	// MigrateDataSchema directly, not through InitDataSchema's dataDDL, so
	// this table must ensure its own existence rather than relying on the
	// DDL constant below having run first).
	if _, err := d.Exec(schemaMetaDDL); err != nil {
		return fmt.Errorf("create schema_meta table: %w", err)
	}

	// Ensure the L1 recall_edges table on every open — additive, so a data.db
	// written by an older rekal (which never ran the newer dataDDL) gains it in
	// place. IF NOT EXISTS makes this idempotent for fresh stores too.
	if _, err := d.Exec(recallEdgesDDL); err != nil {
		return fmt.Errorf("create recall_edges table: %w", err)
	}

	version, err := readSchemaVersion(d, "schema_meta")
	if err != nil {
		return err
	}
	if version > CurrentDataSchemaVersion {
		return &SchemaVersionError{DBName: "data.db", DBVersion: version, MaxVersion: CurrentDataSchemaVersion}
	}

	// Migration: add source column if missing (existing DBs pre-multi-agent).
	if err := addColumnIfMissing(d, "sessions", "source", `VARCHAR NOT NULL DEFAULT 'claude'`); err != nil {
		return err
	}
	// Migration: team/workflow metadata for Claude Code teammates and
	// dynamic workflows (existing DBs pre-subagent-indexing).
	if err := addColumnIfMissing(d, "sessions", "team_name", "VARCHAR"); err != nil {
		return err
	}
	if err := addColumnIfMissing(d, "sessions", "workflow_name", "VARCHAR"); err != nil {
		return err
	}
	// Migration: subagent meta.json sidecar fields (agentType, description,
	// spawnDepth — the real observed shape, see claude.go's subagentMeta doc;
	// existing DBs predate this capture).
	if err := addColumnIfMissing(d, "sessions", "agent_type", "VARCHAR"); err != nil {
		return err
	}
	if err := addColumnIfMissing(d, "sessions", "description", "VARCHAR"); err != nil {
		return err
	}
	if err := addColumnIfMissing(d, "sessions", "spawn_depth", "INTEGER"); err != nil {
		return err
	}
	// Migration: remember which session a captured transcript produced, so a
	// later checkpoint of the same (still-growing) transcript appends to it
	// instead of storing the whole conversation again. checkpoint_state is
	// local-only and never wired, which is why the mapping lives here rather
	// than overloading sessions.session_hash — that column's content-hash
	// meaning is depended on by cross-repo local import.
	if err := addColumnIfMissing(d, "checkpoint_state", "session_id", "VARCHAR"); err != nil {
		return err
	}

	return writeSchemaVersion(d, "schema_meta", CurrentDataSchemaVersion)
}

// InitIndexSchema creates the index DB tables if they do not exist.
// Index DB is derived — can be dropped and rebuilt from data DB.
func InitIndexSchema(d *sql.DB) error {
	if indexDDL == "" {
		return nil
	}
	// Migrate before running the DDL: on an index DB created by an older
	// rekal the CREATE TABLE statements are no-ops, and the CREATE INDEX
	// statements reference columns that only exist after migration.
	if err := MigrateIndexSchema(d); err != nil {
		return err
	}
	if _, err := d.Exec(indexDDL); err != nil {
		return err
	}
	// index_state is guaranteed to exist now (created by indexDDL above, or
	// already existed) — stamp the current version so an older rekal
	// opening this index.db later (e.g. after a local downgrade) detects
	// the mismatch instead of silently misreading columns it predates.
	return WriteIndexState(d, "schema_version", strconv.Itoa(CurrentIndexSchemaVersion))
}

// MigrateIndexSchema applies forward-only migrations to an existing index DB.
// The index can always be rebuilt from the data DB, but incremental indexing
// runs against pre-existing index files, so columns are added in place.
//
// index.db is local-only and always rebuildable, so a version mismatch here
// is lower-stakes than data.db's — it can only happen after a local rekal
// downgrade, never from a teammate's data. Still checked, for the same
// reason a clear error beats a confusing one three layers into a query.
func MigrateIndexSchema(d *sql.DB) error {
	// index_state won't exist yet on a brand-new index.db (InitIndexSchema
	// calls this before running indexDDL) — skip the version check/stamp
	// entirely in that case; InitIndexSchema stamps the version itself once
	// indexDDL has created the table.
	stateTableExists, err := tableExists(d, "index_state")
	if err != nil {
		return err
	}
	if stateTableExists {
		version, err := readSchemaVersion(d, "index_state")
		if err != nil {
			return err
		}
		if version > CurrentIndexSchemaVersion {
			return &SchemaVersionError{DBName: "index.db", DBVersion: version, MaxVersion: CurrentIndexSchemaVersion}
		}
	}

	for _, col := range []struct{ name, typ string }{
		{"parent_session_id", "VARCHAR"},
		{"team_name", "VARCHAR"},
		{"workflow_name", "VARCHAR"},
		{"agent_type", "VARCHAR"},
		{"description", "VARCHAR"},
		{"spawn_depth", "INTEGER"},
		// origin labels a cross-repo local import (rekal index --include*):
		// the source working directory ("repo:/path" or "shell:/path"). NULL
		// for this repo's own sessions and teammate sessions.
		{"origin", "VARCHAR"},
		// facet_text is the session's facet document — distinct tool paths +
		// command prefixes + steering text, capped (PopulateFacetText). The
		// optional facet ranking layer BM25-searches it (weights.facet_boost).
		{"facet_text", "VARCHAR"},
	} {
		if err := addColumnIfMissing(d, "session_facets", col.name, col.typ); err != nil {
			return err
		}
	}

	if stateTableExists {
		return WriteIndexState(d, "schema_version", strconv.Itoa(CurrentIndexSchemaVersion))
	}
	return nil
}

// tableExists reports whether the given table exists in d.
func tableExists(d *sql.DB, table string) (bool, error) {
	var count int
	err := d.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name = $1`, table).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check %s existence: %w", table, err)
	}
	return count > 0, nil
}

// readSchemaVersion reads the 'schema_version' key from a key/value table
// (schema_meta for data.db, index_state for index.db). The table must
// already exist. A missing row (pre-versioning DB, or a table that was just
// created and never stamped) reads as version 0 — the oldest possible,
// always <= any CurrentXSchemaVersion, so old DBs keep opening cleanly.
func readSchemaVersion(d *sql.DB, table string) (int, error) {
	var value string
	err := d.QueryRow(`SELECT value FROM ` + table + ` WHERE key = 'schema_version'`).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("read schema_version from %s: %w", table, err)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid schema_version value %q in %s: %w", value, table, err)
	}
	return n, nil
}

// writeSchemaVersion stamps the 'schema_version' key/value row. The table
// must already exist.
func writeSchemaVersion(d *sql.DB, table string, version int) error {
	_, err := d.Exec(
		`INSERT INTO `+table+` (key, value) VALUES ('schema_version', $1)
		 ON CONFLICT (key) DO UPDATE SET value = $1`,
		strconv.Itoa(version),
	)
	if err != nil {
		return fmt.Errorf("write schema_version to %s: %w", table, err)
	}
	return nil
}

// schemaMetaDDL creates data.db's key/value metadata table (schema_meta),
// mirroring index.db's existing index_state table. Only holds
// 'schema_version' today; a plain key/value shape leaves room for more
// without another migration.
const schemaMetaDDL = `
CREATE TABLE IF NOT EXISTS schema_meta (
	key             VARCHAR PRIMARY KEY,
	value           VARCHAR NOT NULL
);
`

// recallEdgesDDL is the L1 recall citation graph: one row per session an agent
// reached (recalled or drilled) while working. It is the permanent, append-only
// record; the derived index.db.session_reach aggregate is rebuilt from it.
// Local-only by design — deliberately NOT serialized to the codec/wire (like
// checkpoint_state), so it never touches the git transport.
//
// It is ensured in MigrateDataSchema (not just dataDDL) because existing stores
// only run migrations on open, never the full DDL — a new table added to
// dataDDL alone would never appear on a data.db written by an older rekal. The
// CREATE ... IF NOT EXISTS keeps it additive: no schema-version bump. See
// docs/design/recall-graph.md.
const recallEdgesDDL = `
CREATE TABLE IF NOT EXISTS recall_edges (
	id                 VARCHAR PRIMARY KEY,
	ts                 TIMESTAMP NOT NULL,
	kind               VARCHAR NOT NULL,
	query              VARCHAR,
	target_session_id  VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_re_target ON recall_edges(target_session_id);
`

// addColumnIfMissing adds a column to a table when the table exists but the
// column does not. A missing table is a no-op — the CREATE TABLE DDL will
// create it with the full current column set.
//
// DuckDB's ALTER TABLE ADD COLUMN rejects NOT NULL ("Adding columns with
// constraints not yet supported"), so it is stripped from colType before
// adding — DuckDB still backfills existing rows from the DEFAULT. NOT NULL
// is not reapplied afterward: ALTER COLUMN ... SET NOT NULL fails once other
// tables hold a foreign key into this one ("Cannot alter entry ... because
// there are entries that depend on it"), which is always true here by the
// time a migration runs. The column stays nullable on migrated (pre-existing)
// DBs; new DBs get NOT NULL from the CREATE TABLE DDL directly.
func addColumnIfMissing(d *sql.DB, table, column, colType string) error {
	var tables int
	err := d.QueryRow(`SELECT count(*) FROM information_schema.tables
		WHERE table_name = $1`, table).Scan(&tables)
	if err != nil || tables == 0 {
		return nil //nolint:nilerr // best-effort check; later DDL surfaces real problems
	}

	var count int
	err = d.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2`, table, column).Scan(&count)
	if err != nil || count > 0 {
		return nil //nolint:nilerr // best-effort check; later DDL surfaces real problems
	}

	addType := strings.TrimSpace(strings.Replace(colType, "NOT NULL", "", 1))
	_, err = d.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + addType)
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
	branch            VARCHAR,
	source            VARCHAR NOT NULL DEFAULT 'claude',
	team_name         VARCHAR,
	workflow_name     VARCHAR,
	agent_type        VARCHAR,
	description       VARCHAR,
	spawn_depth       INTEGER
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
	file_hash   VARCHAR NOT NULL,
	session_id  VARCHAR
);

CREATE TABLE IF NOT EXISTS schema_meta (
	key         VARCHAR PRIMARY KEY,
	value       VARCHAR NOT NULL
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
	session_id        VARCHAR PRIMARY KEY,
	user_email        VARCHAR,
	git_branch        VARCHAR,
	actor_type        VARCHAR NOT NULL,
	agent_id          VARCHAR,
	captured_at       TIMESTAMP NOT NULL,
	turn_count        INTEGER NOT NULL DEFAULT 0,
	tool_call_count   INTEGER NOT NULL DEFAULT 0,
	file_count        INTEGER NOT NULL DEFAULT 0,
	checkpoint_id     VARCHAR,
	git_sha           VARCHAR,
	parent_session_id VARCHAR,
	team_name         VARCHAR,
	workflow_name     VARCHAR,
	agent_type        VARCHAR,
	description       VARCHAR,
	spawn_depth       INTEGER,
	origin            VARCHAR,
	facet_text        VARCHAR
);
CREATE INDEX IF NOT EXISTS idx_sf_email ON session_facets(user_email);
CREATE INDEX IF NOT EXISTS idx_sf_actor ON session_facets(actor_type);
CREATE INDEX IF NOT EXISTS idx_sf_branch ON session_facets(git_branch);
CREATE INDEX IF NOT EXISTS idx_sf_sha ON session_facets(git_sha);
CREATE INDEX IF NOT EXISTS idx_sf_parent ON session_facets(parent_session_id);
CREATE INDEX IF NOT EXISTS idx_sf_team ON session_facets(team_name);
CREATE INDEX IF NOT EXISTS idx_sf_workflow ON session_facets(workflow_name);

CREATE TABLE IF NOT EXISTS file_cooccurrence (
	file_a          VARCHAR NOT NULL,
	file_b          VARCHAR NOT NULL,
	count           INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (file_a, file_b)
);

CREATE TABLE IF NOT EXISTS session_embeddings (
	session_id      VARCHAR NOT NULL,
	embedding       FLOAT[],
	model           VARCHAR NOT NULL,
	generated_at    TIMESTAMP NOT NULL,
	PRIMARY KEY (session_id, model)
);

CREATE TABLE IF NOT EXISTS index_state (
	key             VARCHAR PRIMARY KEY,
	value           VARCHAR NOT NULL
);
`
