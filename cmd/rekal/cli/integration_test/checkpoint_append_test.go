//go:build integration

package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// sessionDirFor returns (creating if needed) the agent session directory the
// adapters discover for this repo.
func sessionDirFor(t *testing.T, repoDir string) string {
	t.Helper()
	dir := session.FindSessionDir(repoDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// turnLine builds one Claude Code transcript line with the given user text.
func turnLine(text, ts string) string {
	return `{"type":"user","parentMessageId":"","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"` +
		text + `"}]},"timestamp":"` + ts + `","gitBranch":"main"}` + "\n"
}

// transcript builds a transcript whose Nth line is the Nth text.
func transcript(texts ...string) string {
	var b strings.Builder
	b.WriteString(`{"type":"summary","sessionId":"append-001","totalCost":0.01,"totalDuration":10}` + "\n")
	for i, tx := range texts {
		b.WriteString(turnLine(tx, "2026-02-25T10:0"+string(rune('0'+i))+":00Z"))
	}
	return b.String()
}

// openStore opens the test repo's data.db read-only for assertions.
func openStore(t *testing.T, repoDir string) *sql.DB {
	t.Helper()
	d, err := sql.Open("duckdb", filepath.Join(repoDir, ".rekal", "data.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open data.db: %v", err)
	}
	return d
}

func countRows(t *testing.T, d *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := d.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return n
}

// TestCheckpoint_AppendsInsteadOfDuplicating is the regression suite for the
// duplicate-session bug: capture used to key dedup on the transcript's content
// hash, so a conversation that grew between commits was stored again in full.
//
// It covers the three shapes a transcript can take on re-capture — grown,
// unchanged, and rewritten — because only the first may append. The other two
// must leave earlier turns exactly as they were: the ledger is append-only, and
// a wrong append would splice unrelated turns into recorded history.
func TestCheckpoint_AppendsInsteadOfDuplicating(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	sessionDir := sessionDirFor(t, env.RepoDir)
	path := filepath.Join(sessionDir, "grow.jsonl")

	write := func(content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write transcript: %v", err)
		}
	}

	// --- capture 1: three turns ---
	write(transcript("first thing", "second thing", "third thing"))
	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 1: %v", err)
	}
	d := openStore(t, env.RepoDir)
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 1 {
		t.Fatalf("after first capture: %d sessions, want 1", got)
	}
	if got := countRows(t, d, "SELECT count(*) FROM turns"); got != 3 {
		t.Fatalf("after first capture: %d turns, want 3", got)
	}
	var firstID string
	if err := d.QueryRow("SELECT id FROM sessions").Scan(&firstID); err != nil {
		t.Fatalf("read session id: %v", err)
	}
	d.Close() //nolint:errcheck

	// --- capture 2: the conversation grew ---
	write(transcript("first thing", "second thing", "third thing", "fourth thing"))
	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 2: %v", err)
	}
	d = openStore(t, env.RepoDir)
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 1 {
		t.Errorf("growth created %d sessions, want 1 — the conversation must append, not duplicate", got)
	}
	if got := countRows(t, d, "SELECT count(*) FROM turns"); got != 4 {
		t.Errorf("after growth: %d turns, want 4 (three kept, one appended)", got)
	}
	var sameID string
	if err := d.QueryRow("SELECT id FROM sessions").Scan(&sameID); err != nil {
		t.Fatalf("read session id: %v", err)
	}
	if sameID != firstID {
		t.Errorf("session id changed on append: %s -> %s", firstID, sameID)
	}
	// Turn indices must stay contiguous, or drill-down paging breaks.
	if got := countRows(t, d, "SELECT count(DISTINCT turn_index) FROM turns"); got != 4 {
		t.Errorf("turn_index not contiguous after append: %d distinct, want 4", got)
	}
	d.Close() //nolint:errcheck

	// --- capture 3: nothing changed ---
	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 3: %v", err)
	}
	d = openStore(t, env.RepoDir)
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 1 {
		t.Errorf("unchanged transcript created %d sessions, want 1", got)
	}
	if got := countRows(t, d, "SELECT count(*) FROM turns"); got != 4 {
		t.Errorf("unchanged transcript changed turn count to %d, want 4", got)
	}
	d.Close() //nolint:errcheck

	// --- capture 4: the transcript was rewritten, not extended ---
	// This must NOT append: the stored turns are no longer a prefix, so the
	// only safe move is a new session that leaves history untouched.
	write(transcript("a completely different opening", "and another turn"))
	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 4: %v", err)
	}
	d = openStore(t, env.RepoDir)
	defer d.Close() //nolint:errcheck
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 2 {
		t.Errorf("rewritten transcript produced %d sessions, want 2 — it must not append onto the old record", got)
	}
	// The original four turns must survive verbatim.
	if got := countRows(t, d,
		"SELECT count(*) FROM turns WHERE session_id = '"+firstID+"'"); got != 4 {
		t.Errorf("original session now has %d turns, want 4 — history was mutated", got)
	}
}

// TestCheckpoint_SessionHashStaysContentAddressed pins the contract cross-repo
// local import depends on: sessions.session_hash is the transcript's content
// hash. The append mapping deliberately lives in checkpoint_state (local-only,
// never wired) rather than overloading this column, because local_import.go
// dedups by comparing session.ContentHash against these values — repurposing it
// would silently re-import every session this repo already captured.
func TestCheckpoint_SessionHashStaysContentAddressed(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	sessionDir := sessionDirFor(t, env.RepoDir)
	path := filepath.Join(sessionDir, "hash.jsonl")
	content := transcript("only turn")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	d := openStore(t, env.RepoDir)
	defer d.Close() //nolint:errcheck

	var stored string
	if err := d.QueryRow("SELECT session_hash FROM sessions").Scan(&stored); err != nil {
		t.Fatalf("read session_hash: %v", err)
	}
	if want := sha256Hex([]byte(content)); stored != want {
		t.Errorf("session_hash = %q, want the transcript's content hash %q", stored, want)
	}

	// And the append mapping is recorded where it belongs.
	var mapped sql.NullString
	if err := d.QueryRow("SELECT session_id FROM checkpoint_state WHERE file_path = $1", path).Scan(&mapped); err != nil {
		t.Fatalf("read checkpoint_state.session_id: %v", err)
	}
	if !mapped.Valid || mapped.String == "" {
		t.Error("checkpoint_state.session_id not recorded — the next capture would duplicate instead of appending")
	}
}
