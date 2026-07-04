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
