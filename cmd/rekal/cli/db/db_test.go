package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// TestOpen_SingleConnectionPool documents the go-duckdb requirement that
// each *sql.DB use at most one native connection — MaxOpenConns > 1 races
// on WAL auto-checkpoint and can SIGSEGV inside duckdb_pending_prepared.
func TestOpen_SingleConnectionPool(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()

	for name, d := range map[string]*sql.DB{"data": dataDB, "index": indexDB} {
		if got := d.Stats().MaxOpenConnections; got != 1 {
			t.Errorf("%s MaxOpenConnections = %d, want 1", name, got)
		}
	}
}

// TestOpen_ConcurrentQueriesSerialize exercises the single-connection pool
// under parallel readers/writers. Without MaxOpenConns(1) this pattern is
// how go-duckdb hits WAL-checkpoint FATAL/SIGSEGV; with the cap, database/sql
// serializes and the workload must complete cleanly.
func TestOpen_ConcurrentQueriesSerialize(t *testing.T) {
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
	if err := InsertSession(d, "s1", "", "hash-s1", "human", "", "dev@example.com", "main", "2026-07-18T00:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	if err := InsertTurn(d, "t1", "s1", 0, "human", "hello", "2026-07-18T00:00:00Z"); err != nil {
		t.Fatalf("InsertTurn: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				turns, err := QueryTurns(d, "s1")
				if err != nil {
					errCh <- err
					return
				}
				if len(turns) != 1 {
					errCh <- fmt.Errorf("QueryTurns len=%d, want 1", len(turns))
					return
				}
				if _, err := d.Exec(`UPDATE sessions SET branch = 'main' WHERE id = 's1'`); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
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

// TestQueryTurnsPage_LegacySummaryReclassified covers the back-compat path
// for the "summary" role: data.db rows written before the role existed carry
// compaction summaries as role "human" (data.db is append-only, never
// rewritten), and queryTurnsPageFrom reads roles through the
// SummaryFingerprint reclassification so filtering and display match the
// index side.
func TestQueryTurnsPage_LegacySummaryReclassified(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
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

	if err := InsertSession(d, "s1", "", "hash1", "human", "", "a@b.c", "main", "2026-07-11T00:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	// A legacy compaction summary stored as role "human", and a real human turn.
	legacy := SummaryFingerprint + " that ran out of context. Summary of prior work."
	if err := InsertTurn(d, "t1", "s1", 0, "human", legacy, ""); err != nil {
		t.Fatalf("InsertTurn: %v", err)
	}
	if err := InsertTurn(d, "t2", "s1", 1, "human", "continue the refactor", ""); err != nil {
		t.Fatalf("InsertTurn: %v", err)
	}

	// --role summary finds the legacy row.
	rows, total, err := QueryTurnsPage(d, "s1", TurnPageOptions{Role: "summary"})
	if err != nil {
		t.Fatalf("QueryTurnsPage(summary): %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Role != "summary" {
		t.Fatalf("summary filter: total=%d rows=%+v", total, rows)
	}

	// --role human no longer matches it, but keeps the real human turn.
	rows, total, err = QueryTurnsPage(d, "s1", TurnPageOptions{Role: "human"})
	if err != nil {
		t.Fatalf("QueryTurnsPage(human): %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Content != "continue the refactor" {
		t.Fatalf("human filter: total=%d rows=%+v", total, rows)
	}

	// Unfiltered reads show the reclassified role too.
	rows, _, err = QueryTurnsPage(d, "s1", TurnPageOptions{})
	if err != nil {
		t.Fatalf("QueryTurnsPage(all): %v", err)
	}
	if len(rows) != 2 || rows[0].Role != "summary" || rows[1].Role != "human" {
		t.Fatalf("unfiltered roles: %+v", rows)
	}

	// The reclassification is scoped to source 'claude': a Codex session
	// whose human turn happens to start with the same text keeps its role.
	if err := InsertSession(d, "s2", "", "hash2", "human", "", "a@b.c", "main", "2026-07-11T00:00:00Z", "codex"); err != nil {
		t.Fatalf("InsertSession codex: %v", err)
	}
	if err := InsertTurn(d, "t3", "s2", 0, "human", legacy, ""); err != nil {
		t.Fatalf("InsertTurn codex: %v", err)
	}
	rows, total, err = QueryTurnsPage(d, "s2", TurnPageOptions{Role: "summary"})
	if err != nil {
		t.Fatalf("QueryTurnsPage(codex summary): %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("codex session must have no summary turns: total=%d rows=%+v", total, rows)
	}
	rows, total, err = QueryTurnsPage(d, "s2", TurnPageOptions{Role: "human"})
	if err != nil {
		t.Fatalf("QueryTurnsPage(codex human): %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Role != "human" {
		t.Fatalf("codex turn must stay human: total=%d rows=%+v", total, rows)
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
