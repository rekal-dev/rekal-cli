package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenData_CreateAndPing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestOpenIndex_CreateAndPing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestInitDataSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer db.Close()

	if err := InitDataSchema(db); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// Verify tables exist.
	tables := []string{"sessions", "checkpoints", "files_touched", "checkpoint_sessions", "turns", "tool_calls"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s should exist: %v", table, err)
		}
	}
}

func TestInitIndexSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer db.Close()

	// Index schema is placeholder — just verify no error on empty DDL.
	if err := InitIndexSchema(db); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
}

// TestWrapOpenError_LockConflict verifies the exact DuckDB error text
// observed when a second process opens a file another process already has
// open (confirmed by manually reproducing the scenario across two real OS
// processes: "IO Error: Could not set lock on file ...: Conflicting lock is
// held in ... (PID N) by user ...") is translated into a clear,
// actionable message instead of surfacing DuckDB's wording verbatim.
func TestWrapOpenError_LockConflict(t *testing.T) {
	t.Parallel()

	raw := fmt.Errorf(`database/sql/driver: could not connect to database: duckdb error: IO Error: Could not set lock on file "/tmp/data.db": Conflicting lock is held in /usr/local/bin/rekal (PID 253) by user frank. See also https://duckdb.org/docs/connect/concurrency`)

	got := wrapOpenError("/tmp/data.db", raw)
	if !strings.Contains(got.Error(), "another rekal process is already using") {
		t.Errorf("wrapOpenError message = %q, want it to mention another rekal process", got.Error())
	}
	if !errors.Is(got, raw) && !strings.Contains(got.Error(), raw.Error()) {
		t.Errorf("wrapOpenError should preserve the original error for debugging, got %q", got.Error())
	}
}

// TestWrapOpenError_OtherErrorsPassThrough ensures an unrelated failure
// (corrupt file, disk full, ...) is not misreported as a lock conflict.
func TestWrapOpenError_OtherErrorsPassThrough(t *testing.T) {
	t.Parallel()

	raw := fmt.Errorf("duckdb error: IO Error: disk full")
	got := wrapOpenError("/tmp/data.db", raw)
	if strings.Contains(got.Error(), "another rekal process") {
		t.Errorf("wrapOpenError misclassified a non-lock error as a lock conflict: %q", got.Error())
	}
}

// TestQuerySessionContentByIDs_ZeroTurnSession: a session with tool calls but
// no turns aggregates to NULL — the batch must skip it, not error out (it
// used to fail the whole incremental index update).
func TestQuerySessionContentByIDs_ZeroTurnSession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer d.Close()
	if err := InitIndexSchema(d); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}

	if _, err := d.Exec(
		`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
		 VALUES ('t1', 'with-turns', 0, 'human', 'hello world', '')`,
	); err != nil {
		t.Fatalf("seed turn: %v", err)
	}

	got, err := QuerySessionContentByIDs(d, []string{"with-turns", "no-turns"})
	if err != nil {
		t.Fatalf("QuerySessionContentByIDs: %v", err)
	}
	if got["with-turns"] != "hello world" {
		t.Errorf("with-turns content: %q", got["with-turns"])
	}
	if _, ok := got["no-turns"]; ok {
		t.Error("zero-turn session should be absent from the result")
	}
}

// TestQuerySessionIDByHash covers checkpoint's subagent→trunk linking
// fallback: when the trunk was captured in an earlier run, the subagent finds
// its parent row by the trunk file's content hash — most recent capture wins.
func TestQuerySessionIDByHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close()
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// Two captures of the same content hash at different times.
	if err := InsertSession(d, "older", "", "same-hash", "human", "", "dev@x.com", "main", "2026-01-01T10:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession older: %v", err)
	}
	if err := InsertSession(d, "newer", "", "same-hash", "human", "", "dev@x.com", "main", "2026-02-01T10:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession newer: %v", err)
	}

	id, err := QuerySessionIDByHash(d, "same-hash")
	if err != nil {
		t.Fatalf("QuerySessionIDByHash: %v", err)
	}
	if id != "newer" {
		t.Fatalf("id = %q, want the most recently captured (newer)", id)
	}

	if _, err := QuerySessionIDByHash(d, "absent"); err == nil {
		t.Fatal("expected error for an unknown hash")
	}
}

// TestInsertToolCall_UTF8Constraint documents the invariant behind the
// checkpoint "could not bind parameter" bug: DuckDB rejects a VARCHAR bind of
// invalid UTF-8, so scrub.SanitizeText must run before insert. A valid string
// inserts fine; the raw invalid one is what used to crash checkpoints.
func TestInsertToolCall_UTF8Constraint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close()
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}
	if err := InsertSession(d, "s1", "", "h1", "human", "", "a@b.c", "main", "2026-01-01T00:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	// Invalid UTF-8 in cmd_prefix — the exact shape a mid-rune truncation
	// produced — must be rejected by DuckDB (that's why we sanitize upstream).
	badErr := InsertToolCall(d, "tc-bad", "s1", 0, "Bash", "", "run \xc3")
	if badErr == nil {
		t.Fatal("expected DuckDB to reject invalid-UTF-8 cmd_prefix bind")
	}

	// The sanitized equivalent inserts cleanly.
	if err := InsertToolCall(d, "tc-ok", "s1", 1, "Bash", "", "run �"); err != nil {
		t.Fatalf("sanitized cmd_prefix should insert: %v", err)
	}
}
