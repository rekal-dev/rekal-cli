//go:build integration

package integration

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSync_RoundTrip_GrownSessionArrivesOnce is the end-to-end proof for the
// duplicate-session fix, across the seam that actually matters: another machine.
//
// It exercises the shape every earlier test skipped — a conversation captured at
// two commits, having grown in between — and follows it all the way out to the
// wire and back into a second repo. Three things could each reintroduce the bug
// independently, and only a round trip catches all three: capture could store the
// conversation twice, export could ship both copies, and import could accept them.
//
// Asserts effect, not exit codes. Every command here returns success in the buggy
// world too; only the session count tells the truth.
func TestSync_RoundTrip_GrownSessionArrivesOnce(t *testing.T) {
	author := NewTestEnv(t)
	author.Init()

	bare := t.TempDir()
	bare, _ = filepath.EvalSymlinks(bare)
	initBareRemote(t, bare)
	if err := exec.Command("git", "-C", author.RepoDir, "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	if err := os.WriteFile(filepath.Join(author.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, author.RepoDir, "initial")
	// The mainline must exist on the remote: the merged-only gate shares only
	// checkpoints whose commits landed there.
	pushMain(t, author.RepoDir)

	sessionDir := sessionDirFor(t, author.RepoDir)
	path := filepath.Join(sessionDir, "conv.jsonl")

	// --- commit 1: the conversation so far ---
	if err := os.WriteFile(path, []byte(transcript("start of the work", "second turn")), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(author.RepoDir, "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, author.RepoDir, "first change")
	pushMain(t, author.RepoDir)
	if _, _, err := author.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 1: %v", err)
	}

	// --- commit 2: same conversation, now longer ---
	if err := os.WriteFile(path, []byte(transcript("start of the work", "second turn", "third turn")), 0o644); err != nil {
		t.Fatalf("grow transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(author.RepoDir, "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, author.RepoDir, "second change")
	pushMain(t, author.RepoDir)
	if _, _, err := author.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 2: %v", err)
	}

	// The author's own ledger holds one conversation, not two.
	d := openStore(t, author.RepoDir)
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 1 {
		t.Fatalf("author has %d sessions, want 1 — growth must append", got)
	}
	if got := countRows(t, d, "SELECT count(*) FROM turns"); got != 3 {
		t.Errorf("author has %d turns, want 3", got)
	}
	d.Close() //nolint:errcheck

	if _, stderr, err := author.RunCLI("push"); err != nil {
		t.Fatalf("push: %v\n%s", err, stderr)
	}

	// --- a teammate clones and syncs ---
	peerRoot := t.TempDir()
	peerRoot, _ = filepath.EvalSymlinks(peerRoot)
	peerDir := filepath.Join(peerRoot, "clone")
	if out, err := exec.Command("git", "clone", "-q", bare, peerDir).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	// A distinct identity, or sync would treat the author's rekal/<email> branch
	// as the peer's own and skip importing it.
	for _, kv := range [][2]string{{"user.email", "peer@rekal.dev"}, {"user.name", "peer"}} {
		if err := exec.Command("git", "-C", peerDir, "config", kv[0], kv[1]).Run(); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}
	peer := NewTestEnvAt(t, peerDir)
	peer.Init()
	if _, se, err := peer.RunCLI("sync"); err != nil {
		t.Fatalf("peer sync: %v\n%s", err, se)
	}

	// Team sessions land in the derived index, not the peer's own data.db —
	// that is what recall reads, so it is what has to be right.
	p, err := sql.Open("duckdb", filepath.Join(peerDir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open peer index: %v", err)
	}
	defer p.Close() //nolint:errcheck

	if got := countRows(t, p, "SELECT count(*) FROM session_facets"); got != 1 {
		t.Errorf("peer received %d sessions, want 1 — the wire must not carry the same conversation twice", got)
	}
	// The peer must get the *complete* conversation, not the earlier snapshot:
	// dropping duplicates has to keep the longest, never the first.
	if got := countRows(t, p, "SELECT count(*) FROM turns_ft"); got != 3 {
		t.Errorf("peer received %d turns, want 3 — the longest capture must survive, not a truncated one", got)
	}
	var last string
	if err := p.QueryRow("SELECT content FROM turns_ft ORDER BY turn_index DESC LIMIT 1").Scan(&last); err != nil {
		t.Fatalf("read last turn: %v", err)
	}
	if !strings.Contains(last, "third turn") {
		t.Errorf("peer's last turn = %q, want the turn added at the second commit", last)
	}
}

// pushMain publishes the working branch so the merged-only gate can see that the
// commits landed on the mainline.
func pushMain(t *testing.T, repoDir string) {
	t.Helper()
	branch, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}
	b := strings.TrimSpace(string(branch))
	cmd := exec.Command("git", "-C", repoDir, "push", "--no-verify", "-q", "origin", b)
	cmd.Env = append(os.Environ(), "HOME=/nonexistent", "PATH=/usr/bin:/bin")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("push %s: %v\n%s", b, err, out)
	}
}
