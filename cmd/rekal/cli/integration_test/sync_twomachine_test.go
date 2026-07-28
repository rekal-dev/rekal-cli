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

// newMachine clones the shared remote under the SAME identity as any other
// machine created with the same email — so both push to one rekal/<email>
// branch. That single shared branch is what makes this different from the
// multi-peer shape, where every author owns a branch nobody else writes.
func newMachine(t *testing.T, bare, email, name string) *peerRepo {
	t.Helper()
	return newPeer(t, bare, email, name)
}

// pullMain brings a machine's main branch up to date, as anyone working from a
// second machine would before starting again.
func pullMain(t *testing.T, repoDir string) {
	t.Helper()
	branch, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}
	b := strings.TrimSpace(string(branch))
	cmd := exec.Command("git", "-C", repoDir, "pull", "--rebase", "-q", "origin", b)
	cmd.Env = append(os.Environ(),
		"HOME=/nonexistent", "PATH=/usr/bin:/bin",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pull %s in %s: %v\n%s", b, repoDir, err, out)
	}
}

// newSharedRemote builds a bare remote with one commit on main, which the
// machines below clone.
func newSharedRemote(t *testing.T) string {
	t.Helper()
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
	return bare
}

// TestSync_TwoMachines_SameIdentity covers one person working from two machines.
//
// The wire body is cumulative along the *local* orphan branch: export reads the
// body at the local tip and appends to it. So a machine whose branch is behind
// builds a body missing everything the other machine pushed, and its push is
// rejected. Nothing used to move that ref — EnsureOrphanBranch seeds it from the
// remote only at first creation — so the second machine was stuck with two ways
// out, a manual reset or a --force that deletes the other machine's
// conversations from the shared branch.
//
// The push path now fast-forwards the local branch first when the remote
// strictly contains it, which turns this from a dead end into the ordinary case.
func TestSync_TwoMachines_SameIdentity(t *testing.T) {
	bare := newSharedRemote(t)
	const email = "solo@dev.example"

	laptop := newMachine(t, bare, email, "laptop")
	laptop.contribute(t, "laptop.jsonl",
		[]string{"laptop opening turn", "laptop decided to drop the queue"}, "laptop work")

	// The desktop clones after that push, so its orphan branch is seeded from
	// the remote and its own push is an ordinary fast-forward.
	desktop := newMachine(t, bare, email, "desktop")
	desktop.contribute(t, "desktop.jsonl",
		[]string{"desktop opening turn", "desktop rewrote the parser"}, "desktop work")

	// Now the laptop works again. This is the case that used to be a dead end:
	// its local orphan branch still points at its own first push, so the remote
	// has moved on without it. Export appends to that stale tip, producing a
	// body with no trace of the desktop's work, and the push is rejected — the
	// laptop cannot contribute again without a manual reset or a --force that
	// would delete the desktop's conversation from the shared branch.
	pullMain(t, laptop.dir)
	laptop.contribute(t, "laptop2.jsonl",
		[]string{"laptop second opening", "laptop later fixed the retry loop"}, "laptop more work")

	// Read it back the way a third machine would: fresh clone, sync --self.
	reader := newMachine(t, bare, email, "reader")
	if _, stderr, err := reader.env.RunCLI("sync", "--self"); err != nil {
		t.Fatalf("reader sync --self: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	for _, needle := range []string{
		"laptop decided to drop the queue",
		"desktop rewrote the parser",
		"laptop later fixed the retry loop",
	} {
		var n int
		if err := idx.QueryRow(
			`SELECT count(*) FROM turns_ft WHERE content LIKE '%' || $1 || '%'`, needle,
		).Scan(&n); err != nil {
			t.Fatalf("query turns_ft: %v", err)
		}
		if n == 0 {
			t.Errorf("%q is missing — a machine pushing from a stale branch either lost its own work or overwrote the other machine's", needle)
		}
	}
}

// TestSync_TwoMachines_GrownSessionIsNotTruncated covers the second half: the
// same conversation arriving twice, longer the second time.
//
// One machine pushes a session at one commit, keeps talking, and pushes again.
// The data.db import deduplicated by session id alone and returned early on
// "exists", so the reader kept whatever length arrived first — permanently, and
// no amount of re-syncing recovered it. Unlike the index path, which had already
// learned to keep the longest, this one silently truncated.
func TestSync_TwoMachines_GrownSessionIsNotTruncated(t *testing.T) {
	bare := newSharedRemote(t)
	const email = "grow@dev.example"

	author := newMachine(t, bare, email, "author")
	author.contribute(t, "s.jsonl",
		[]string{"shared opening", "the first half"}, "first push")

	reader := newMachine(t, bare, email, "reader")
	if _, stderr, err := reader.env.RunCLI("sync", "--self"); err != nil {
		t.Fatalf("reader first sync: %v\n%s", err, stderr)
	}

	// The author keeps talking in the SAME transcript and pushes again.
	sessionDir := sessionDirFor(t, author.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, "s.jsonl"),
		[]byte(transcript("shared opening", "the first half", "the decisive second half")), 0o644); err != nil {
		t.Fatalf("grow transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(author.dir, "more.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, author.dir, "second push")
	pushMain(t, author.dir)
	if _, stderr, err := author.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("author checkpoint 2: %v\n%s", err, stderr)
	}
	if _, stderr, err := author.env.RunCLI("push"); err != nil {
		t.Fatalf("author push 2: %v\n%s", err, stderr)
	}

	if _, stderr, err := reader.env.RunCLI("sync", "--self"); err != nil {
		t.Fatalf("reader second sync: %v\n%s", err, stderr)
	}

	data, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "data.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader data.db: %v", err)
	}
	defer data.Close() //nolint:errcheck

	var n int
	if err := data.QueryRow(
		`SELECT count(*) FROM turns WHERE content LIKE '%the decisive second half%'`,
	).Scan(&n); err != nil {
		t.Fatalf("query turns: %v", err)
	}
	if n == 0 {
		t.Error("the appended turn never arrived — the conversation is stuck at the length that happened to sync first, and re-syncing will never fix it")
	}
	if n > 1 {
		t.Errorf("the appended turn arrived %d times — the import duplicated instead of extending", n)
	}

	// And the earlier turns must not have been re-inserted alongside it.
	var opening int
	if err := data.QueryRow(
		`SELECT count(*) FROM turns WHERE content LIKE '%shared opening%'`,
	).Scan(&opening); err != nil {
		t.Fatalf("query opening: %v", err)
	}
	if opening != 1 {
		t.Errorf("opening turn stored %d times, want 1 — the grown frame must append past what exists, not restore the whole transcript", opening)
	}
}
