package db

import (
	"os"
	"path/filepath"
	"testing"
)

func openTempDB(t *testing.T) (string, *testDB) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, nil
}

type testDB struct{}

// oldDataSessionsDDL mirrors the sessions table as created by rekal versions
// before team/workflow metadata (v0.2.x).
const oldDataSessionsDDL = `
CREATE TABLE sessions (
	id                VARCHAR PRIMARY KEY,
	parent_session_id VARCHAR,
	session_hash      VARCHAR NOT NULL,
	captured_at       TIMESTAMP NOT NULL,
	actor_type        VARCHAR NOT NULL DEFAULT 'human',
	agent_id          VARCHAR,
	user_email        VARCHAR,
	branch            VARCHAR,
	source            VARCHAR NOT NULL DEFAULT 'claude'
);`

// oldFacetsDDL mirrors session_facets before parent/team/workflow columns.
const oldFacetsDDL = `
CREATE TABLE session_facets (
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
);`

// TestMigrateDataSchema_OldDataDB simulates opening a data.db written by an
// older rekal (shared via git from a teammate): existing rows must survive,
// new columns must appear, and new-style inserts must work afterwards.
func TestMigrateDataSchema_OldDataDB(t *testing.T) {
	t.Parallel()

	dir, _ := openTempDB(t)
	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close()

	// Old-version schema + a row captured by the old version.
	if _, err := d.Exec(oldDataSessionsDDL); err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	if _, err := d.Exec(
		`INSERT INTO sessions (id, session_hash, captured_at, actor_type, user_email, branch)
		 VALUES ('old1', 'h1', '2026-01-01 10:00:00', 'human', 'greg@example.com', 'main')`,
	); err != nil {
		t.Fatalf("insert old row: %v", err)
	}

	// New version opens the DB: InitDataSchema must migrate in place.
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema on old DB: %v", err)
	}
	// Idempotent: safe to run again.
	if err := MigrateDataSchema(d); err != nil {
		t.Fatalf("MigrateDataSchema (second run): %v", err)
	}

	// Old row intact, new columns readable as NULL/empty.
	var email, team, workflow string
	err = d.QueryRow(
		`SELECT user_email, COALESCE(team_name, ''), COALESCE(workflow_name, '')
		 FROM sessions WHERE id = 'old1'`,
	).Scan(&email, &team, &workflow)
	if err != nil {
		t.Fatalf("read old row after migration: %v", err)
	}
	if email != "greg@example.com" || team != "" || workflow != "" {
		t.Errorf("old row after migration = %q/%q/%q, want greg@example.com//", email, team, workflow)
	}

	// New-style insert with team/workflow metadata works.
	if err := InsertSessionMeta(d, "new1", "old1", "h2", "agent", "researcher-2",
		"frank@example.com", "main", "2026-07-01T10:00:00Z", "claude", "perf-team", "release-flow"); err != nil {
		t.Fatalf("InsertSessionMeta after migration: %v", err)
	}
	var gotTeam, gotWorkflow, gotParent string
	err = d.QueryRow(
		`SELECT COALESCE(team_name,''), COALESCE(workflow_name,''), COALESCE(parent_session_id,'')
		 FROM sessions WHERE id = 'new1'`,
	).Scan(&gotTeam, &gotWorkflow, &gotParent)
	if err != nil {
		t.Fatalf("read new row: %v", err)
	}
	if gotTeam != "perf-team" || gotWorkflow != "release-flow" || gotParent != "old1" {
		t.Errorf("new row = %q/%q/%q, want perf-team/release-flow/old1", gotTeam, gotWorkflow, gotParent)
	}
}

// TestMigrateIndexSchema_OldIndexDB simulates an index.db created by an older
// rekal: incremental indexing must find the new facets columns after migration.
func TestMigrateIndexSchema_OldIndexDB(t *testing.T) {
	t.Parallel()

	dir, _ := openTempDB(t)
	d, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(oldFacetsDDL); err != nil {
		t.Fatalf("create old facets: %v", err)
	}
	if _, err := d.Exec(
		`INSERT INTO session_facets (session_id, actor_type, captured_at)
		 VALUES ('s1', 'human', '2026-01-01 10:00:00')`,
	); err != nil {
		t.Fatalf("insert old facets row: %v", err)
	}

	// InitIndexSchema (as run on open) must add the new columns in place.
	if err := InitIndexSchema(d); err != nil {
		t.Fatalf("InitIndexSchema on old DB: %v", err)
	}
	if err := MigrateIndexSchema(d); err != nil {
		t.Fatalf("MigrateIndexSchema (second run): %v", err)
	}

	var team, workflow, parent string
	err = d.QueryRow(
		`SELECT COALESCE(team_name,''), COALESCE(workflow_name,''), COALESCE(parent_session_id,'')
		 FROM session_facets WHERE session_id = 's1'`,
	).Scan(&team, &workflow, &parent)
	if err != nil {
		t.Fatalf("read old facets row after migration: %v", err)
	}
	if team != "" || workflow != "" || parent != "" {
		t.Errorf("migrated facets row = %q/%q/%q, want all empty", team, workflow, parent)
	}
}

// TestPopulateIndex_OptionalMetadataAcrossAgents verifies that sessions
// without harness metadata (Codex etc.) index cleanly with NULLs while
// sessions with metadata carry it into session_facets — no per-agent
// special-casing.
func TestPopulateIndex_OptionalMetadataAcrossAgents(t *testing.T) {
	t.Parallel()

	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// A Codex session: no team/workflow/parent/agent metadata.
	if err := InsertSession(dataDB, "codex1", "", "h1", "human", "", "frank@example.com", "main", "2026-07-01T10:00:00Z", "codex"); err != nil {
		t.Fatalf("InsertSession codex: %v", err)
	}
	// A Claude workflow transcript with full metadata.
	if err := InsertSessionMeta(dataDB, "wf1", "codex1", "h2", "agent", "step1",
		"frank@example.com", "main", "2026-07-01T11:00:00Z", "claude", "perf-team", "release-flow"); err != nil {
		t.Fatalf("InsertSessionMeta: %v", err)
	}
	for _, sid := range []string{"codex1", "wf1"} {
		if err := InsertTurn(dataDB, sid+":0", sid, 0, "human", "hello from "+sid, ""); err != nil {
			t.Fatalf("InsertTurn %s: %v", sid, err)
		}
	}
	dataDB.Close()

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	var team, workflow, parent string
	if err := indexDB.QueryRow(
		`SELECT COALESCE(team_name,''), COALESCE(workflow_name,''), COALESCE(parent_session_id,'')
		 FROM session_facets WHERE session_id = 'codex1'`,
	).Scan(&team, &workflow, &parent); err != nil {
		t.Fatalf("read codex facets: %v", err)
	}
	if team != "" || workflow != "" || parent != "" {
		t.Errorf("codex facets = %q/%q/%q, want all empty", team, workflow, parent)
	}

	if err := indexDB.QueryRow(
		`SELECT COALESCE(team_name,''), COALESCE(workflow_name,''), COALESCE(parent_session_id,'')
		 FROM session_facets WHERE session_id = 'wf1'`,
	).Scan(&team, &workflow, &parent); err != nil {
		t.Fatalf("read workflow facets: %v", err)
	}
	if team != "perf-team" || workflow != "release-flow" || parent != "codex1" {
		t.Errorf("workflow facets = %q/%q/%q, want perf-team/release-flow/codex1", team, workflow, parent)
	}
}

// TestInsertSession_BackwardCompatibleWrapper ensures the pre-metadata insert
// path still works unchanged (used by import and older call sites).
func TestInsertSession_BackwardCompatibleWrapper(t *testing.T) {
	t.Parallel()

	dir, _ := openTempDB(t)
	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close()
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	if err := InsertSession(d, "s1", "", "h1", "human", "", "frank@example.com", "main", "2026-07-01T10:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	var team string
	if err := d.QueryRow(`SELECT COALESCE(team_name,'') FROM sessions WHERE id='s1'`).Scan(&team); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if team != "" {
		t.Errorf("team_name = %q, want empty", team)
	}
}
