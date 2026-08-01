//go:build integration

package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestSync_BenchFixturesFromAPeerAreDropped pins the purge on the arrival path.
//
// session.SkipCapture keeps RekalBench fixtures out of data.db, but it is a
// guard on capture, not on import: a teammate whose store predates it — or who
// runs the harness on an older binary — already has fixtures in their ledger,
// and the ledger is append-only, so they ride the wire forever. The reader is
// the only one who can decline them.
//
// PopulateIndex runs the same purge, but it runs before a single remote session
// exists in the index, so it only ever sees what local data.db holds. The gap is
// the one PurgeSupersededSessionsWithData had on this path: the pass has to run
// again after the import loop, or it never sees an arrival at all.
func TestSync_BenchFixturesFromAPeerAreDropped(t *testing.T) {
	bare := newSharedRemote(t)

	// A teammate with real work.
	good := newPeer(t, bare, "good@rekal.dev", "good")
	good.contribute(t, "s.jsonl", []string{"how should the wire format handle dictionaries", "preset dictionary, shipped in the binary"}, "good work")

	// A teammate whose ledger predates the capture guard: the session is
	// captured, then a harness prompt is appended straight into data.db — which
	// is what a store written before SkipCapture existed actually looks like —
	// and only then pushed.
	polluted := newPeer(t, bare, "polluted@rekal.dev", "polluted")
	sessionDir := sessionDirFor(t, polluted.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, "s.jsonl"),
		[]byte(transcript("generate the eval queries", "running the generator")), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(polluted.dir, "polluted.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, polluted.dir, "harness run")
	pushMain(t, polluted.dir)
	if _, stderr, err := polluted.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("polluted checkpoint: %v\n%s", err, stderr)
	}
	plantBenchTurn(t, polluted.dir)
	if _, stderr, err := polluted.env.RunCLI("push"); err != nil {
		t.Fatalf("polluted push: %v\n%s", err, stderr)
	}

	reader := newPeer(t, bare, "reader3@rekal.dev", "reader3")
	if _, stderr, err := reader.env.RunCLI("sync"); err != nil {
		t.Fatalf("reader sync: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	if got := countRows(t, idx, `SELECT count(*) FROM turns_ft WHERE content LIKE '%RekalBench%'`); got != 0 {
		t.Errorf("reader's index carries %d harness turn(s), want 0 — a fixture that arrives over the wire must be declined the same way a local one is", got)
	}
	// The real teammate must survive: a purge that took the whole import with it
	// would satisfy the assertion above and be strictly worse than the bug.
	if got := countRows(t, idx, `SELECT count(*) FROM turns_ft WHERE content LIKE '%preset dictionary%'`); got != 1 {
		t.Errorf("good teammate's turn appears %d times, want 1 — dropping fixtures must not cost real memory", got)
	}
}

// plantBenchTurn appends a harness prompt to the newest session in a repo's
// data.db, simulating a ledger written before session.SkipCapture existed.
func plantBenchTurn(t *testing.T, repoDir string) {
	t.Helper()
	data, err := sql.Open("duckdb", filepath.Join(repoDir, ".rekal", "data.db"))
	if err != nil {
		t.Fatalf("open data.db: %v", err)
	}
	defer data.Close() //nolint:errcheck

	var sid string
	var next int
	if err := data.QueryRow(`SELECT id FROM sessions ORDER BY id DESC LIMIT 1`).Scan(&sid); err != nil {
		t.Fatalf("find session: %v", err)
	}
	if err := data.QueryRow(`SELECT coalesce(max(turn_index), -1) + 1 FROM turns WHERE session_id = $1`, sid).Scan(&next); err != nil {
		t.Fatalf("find turn index: %v", err)
	}
	if _, err := data.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content, ts) VALUES ($1, $2, $3, 'human', $4, now())`,
		sid+"-bench", sid, next, "RekalBench: generate labels for the T1 split",
	); err != nil {
		t.Fatalf("plant bench turn: %v", err)
	}
}
