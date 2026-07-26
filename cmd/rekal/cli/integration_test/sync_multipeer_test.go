//go:build integration

package integration

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// peerRepo is one teammate: their own clone, identity, and rekal branch.
type peerRepo struct {
	env   *TestEnv
	dir   string
	email string
}

// newPeer clones the shared remote and initializes rekal under a distinct
// identity, so each teammate pushes to their own rekal/<email> branch.
func newPeer(t *testing.T, bare, email, name string) *peerRepo {
	t.Helper()
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	dir := filepath.Join(root, name)
	if out, err := exec.Command("git", "clone", "-q", bare, dir).CombinedOutput(); err != nil {
		t.Fatalf("clone %s: %v\n%s", name, err, out)
	}
	for _, kv := range [][2]string{{"user.email", email}, {"user.name", name}} {
		if err := exec.Command("git", "-C", dir, "config", kv[0], kv[1]).Run(); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}
	env := NewTestEnvAt(t, dir)
	env.Init()
	return &peerRepo{env: env, dir: dir, email: email}
}

// contribute writes a transcript, commits, checkpoints and pushes it.
func (p *peerRepo) contribute(t *testing.T, file string, turns []string, commitMsg string) {
	t.Helper()
	sessionDir := sessionDirFor(t, p.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, file), []byte(transcript(turns...)), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p.dir, p.email+".go"), []byte("package main // "+commitMsg+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, p.dir, commitMsg)
	pushMain(t, p.dir)
	if _, stderr, err := p.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("%s checkpoint: %v\n%s", p.email, err, stderr)
	}
	if _, stderr, err := p.env.RunCLI("push"); err != nil {
		t.Fatalf("%s push: %v\n%s", p.email, err, stderr)
	}
}

// TestSync_MultiPeer_BranchesStayDistinct exercises the real team shape: several
// teammates, each pushing their own rekal/<email> branch, all imported by one
// reader.
//
// Two hazards only appear with more than one peer. First, imported sessions live
// in the derived index with no data.db row, so the supersession discriminator
// (source, author, parent) is unavailable for them — leaving turn 0 as the only
// signal, and two teammates whose sessions open identically would be collapsed
// into one. Boilerplate first turns are the norm, not the exception. Second, one
// unreadable branch must not abort the others: import warns per branch, and a
// reader with three teammates should not lose all three because one is broken.
func TestSync_MultiPeer_BranchesStayDistinct(t *testing.T) {
	origin := NewTestEnv(t)
	origin.Init()

	bare := t.TempDir()
	bare, _ = filepath.EvalSymlinks(bare)
	initBareRemote(t, bare)
	if err := exec.Command("git", "-C", origin.RepoDir, "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	if err := os.WriteFile(filepath.Join(origin.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, origin.RepoDir, "initial")
	pushMain(t, origin.RepoDir)

	// Three teammates. Every session opens with the same boilerplate turn — the
	// realistic case, and the one that breaks a turn-0-only discriminator. Each
	// then diverges, and each grows across two commits so the append path is
	// exercised per peer.
	const boilerplate = "continue the previous work"
	peers := []struct{ email, name, topic string }{
		{"alice@rekal.dev", "alice", "alice investigates the parser"},
		{"bob@rekal.dev", "bob", "bob rewrites the scheduler"},
		{"carol@rekal.dev", "carol", "carol audits the wire format"},
	}
	for _, spec := range peers {
		p := newPeer(t, bare, spec.email, spec.name)
		p.contribute(t, "s.jsonl", []string{boilerplate, spec.topic}, spec.name+" first")
		p.contribute(t, "s.jsonl", []string{boilerplate, spec.topic, spec.topic + " — continued"}, spec.name+" second")
	}

	// A fourth person reads everyone's memory.
	reader := newPeer(t, bare, "reader@rekal.dev", "reader")
	if _, stderr, err := reader.env.RunCLI("sync"); err != nil {
		t.Fatalf("reader sync: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	// Every teammate's conversation must survive as its own session. Collapsing
	// them would silently erase two people's memory.
	if got := countRows(t, idx, "SELECT count(*) FROM session_facets"); got != len(peers) {
		t.Errorf("reader sees %d sessions, want %d — one per teammate; a shared opening turn must not merge distinct people's work", got, len(peers))
	}

	// And each must be the grown version, not the first snapshot.
	for _, spec := range peers {
		var n int
		if err := idx.QueryRow(
			`SELECT count(*) FROM turns_ft WHERE content LIKE $1`, "%"+spec.topic+" — continued%",
		).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", spec.name, err)
		}
		if n != 1 {
			t.Errorf("%s's later turn appears %d times, want exactly 1 — their conversation must arrive complete and once", spec.name, n)
		}
	}
}

// TestSync_MultiPeer_OneBrokenBranchDoesNotBlockOthers pins the failure mode a
// reader actually cares about: a teammate's branch that cannot be decoded must
// cost that teammate's memory only, not everyone's.
func TestSync_MultiPeer_OneBrokenBranchDoesNotBlockOthers(t *testing.T) {
	origin := NewTestEnv(t)
	origin.Init()

	bare := t.TempDir()
	bare, _ = filepath.EvalSymlinks(bare)
	initBareRemote(t, bare)
	if err := exec.Command("git", "-C", origin.RepoDir, "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	if err := os.WriteFile(filepath.Join(origin.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, origin.RepoDir, "initial")
	pushMain(t, origin.RepoDir)

	good := newPeer(t, bare, "good@rekal.dev", "good")
	good.contribute(t, "s.jsonl", []string{"a real conversation", "with real content"}, "good work")

	// A branch whose wire bytes are garbage.
	broken := t.TempDir()
	broken, _ = filepath.EvalSymlinks(broken)
	brokenDir := filepath.Join(broken, "broken")
	if out, err := exec.Command("git", "clone", "-q", bare, brokenDir).CombinedOutput(); err != nil {
		t.Fatalf("clone broken: %v\n%s", err, out)
	}
	writeCorruptBranch(t, brokenDir, "rekal/broken@rekal.dev")

	reader := newPeer(t, bare, "reader2@rekal.dev", "reader2")
	_, stderr, err := reader.env.RunCLI("sync")
	if err != nil {
		t.Fatalf("sync must not fail because one branch is corrupt: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	if got := countRows(t, idx, "SELECT count(*) FROM session_facets"); got != 1 {
		t.Errorf("reader sees %d sessions, want 1 — the healthy teammate's memory must survive a corrupt sibling branch", got)
	}
}

// writeCorruptBranch publishes an orphan branch carrying invalid wire bytes.
func writeCorruptBranch(t *testing.T, repoDir, branch string) {
	t.Helper()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=x", "GIT_AUTHOR_EMAIL=x@x",
			"GIT_COMMITTER_NAME=x", "GIT_COMMITTER_EMAIL=x@x")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return trimNL(string(out))
	}
	if err := os.WriteFile(filepath.Join(repoDir, "rekal.body"), []byte("NOTAWIREFORMAT"), 0o644); err != nil {
		t.Fatal(err)
	}
	blob := run("hash-object", "-w", filepath.Join(repoDir, "rekal.body"))
	run("read-tree", "--empty")
	run("update-index", "--add", "--cacheinfo", fmt.Sprintf("100644,%s,rekal.body", blob))
	tree := run("write-tree")
	commit := run("commit-tree", tree, "-m", "corrupt wire")
	run("update-ref", "refs/heads/"+branch, commit)
	cmd := exec.Command("git", "-C", repoDir, "push", "--no-verify", "-q", "origin", branch)
	cmd.Env = append(os.Environ(), "HOME=/nonexistent", "PATH=/usr/bin:/bin")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("push corrupt branch: %v\n%s", err, out)
	}
	os.Remove(filepath.Join(repoDir, "rekal.body")) //nolint:errcheck
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
