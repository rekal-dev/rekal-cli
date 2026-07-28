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

// TestPush_StrandedByRewriteStillShips covers the checkpoint whose commit is
// rewritten between capture and its first push.
//
// The post-commit hook anchors a checkpoint to the sha that exists at that
// moment. Amending the message, or rebasing onto an updated mainline before
// merging, replaces that commit — the sha survives as an object but stops being
// reachable from the branch, so the merged-only gate's ancestor test fails and
// keeps failing. The work is on the mainline under a new name; the checkpoint is
// withheld on every future push, with nothing to retry and nothing reported. If
// no later checkpoint happens to cover the same session, that conversation never
// reaches the team at all.
func TestPush_StrandedByRewriteStillShips(t *testing.T) {
	bare := newSharedRemote(t)
	dev := newMachine(t, bare, "rewrite@dev.example", "dev")

	// The default branch name is whatever git init produced here, not
	// necessarily "main".
	mainBranch := currentBranch(t, dev.dir)

	// Work captured on a branch, so the sha can be rewritten before it lands.
	gitRun(t, dev.dir, "checkout", "-q", "-b", "feature")
	sessionDir := sessionDirFor(t, dev.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, "s.jsonl"),
		[]byte(transcript("why the retry loop backs off", "because the gateway rate-limits bursts")), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dev.dir, "feature.go"), []byte("package main // feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, dev.dir, "feature work")
	if _, stderr, err := dev.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint: %v\n%s", err, stderr)
	}

	// Rewrite it — an amended message is the mildest possible case, and it is
	// enough to orphan the sha the checkpoint points at.
	gitRun(t, dev.dir, "commit", "--amend", "-q", "-m", "feature work (reworded)")

	// It lands on the mainline under the new sha.
	gitRun(t, dev.dir, "checkout", "-q", mainBranch)
	gitRun(t, dev.dir, "merge", "-q", "--ff-only", "feature")
	pushMain(t, dev.dir)

	if _, stderr, err := dev.env.RunCLI("push"); err != nil {
		t.Fatalf("push: %v\n%s", err, stderr)
	}

	// The conversation must be on the wire, readable by anyone who syncs.
	reader := newMachine(t, bare, "rewrite@dev.example", "reader")
	if _, stderr, err := reader.env.RunCLI("sync", "--self"); err != nil {
		t.Fatalf("reader sync --self: %v\n%s", err, stderr)
	}
	data, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "data.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader data.db: %v", err)
	}
	defer data.Close() //nolint:errcheck

	var n int
	if err := data.QueryRow(
		`SELECT count(*) FROM turns WHERE content LIKE '%gateway rate-limits bursts%'`,
	).Scan(&n); err != nil {
		t.Fatalf("query turns: %v", err)
	}
	if n == 0 {
		t.Error("the conversation never reached the wire — its commit was rewritten before the first push, so the gate can no longer prove the work merged even though it plainly did")
	}
}

// TestPush_ForkNeverDiscardsAnotherMachine covers the append-only guarantee at
// the point it is easiest to break.
//
// Two machines share one rekal/<email> branch, so a genuine fork makes git
// demand a force — and force is exactly wrong there: each machine's data.db
// holds only its own checkpoints, and sync imports other branches into the
// index and never into data.db, so frames deleted from the branch cannot be
// re-exported by anyone. rekal push offers no way to do it: the wire format is
// append-only, and that is structural rather than a policy with an override.
// --rebuild, the one path that rewrites the body, refuses unless what the
// branch holds is already contained in what it would write.
func TestPush_ForkNeverDiscardsAnotherMachine(t *testing.T) {
	bare := newSharedRemote(t)
	const email = "guard@dev.example"

	// Both machines are set up before either pushes, so each creates its own
	// orphan branch root and the two genuinely fork — no shared history to
	// fast-forward across. This is the ordinary shape when someone sets up two
	// machines in the same afternoon.
	laptop := newMachine(t, bare, email, "laptop")
	desktop := newMachine(t, bare, email, "desktop")

	laptop.contribute(t, "laptop.jsonl",
		[]string{"laptop opening", "laptop found the deadlock"}, "laptop work")
	pullMain(t, desktop.dir)
	desktop.contribute(t, "desktop.jsonl",
		[]string{"desktop opening", "desktop rewrote the parser"}, "desktop work")

	// There is no rekal-level force at all.
	if _, stderr, err := desktop.env.RunCLI("push", "--force"); err == nil {
		t.Errorf("push --force was accepted; the append-only guarantee must not have an override flag: %s", stderr)
	}
	// And the one path that rewrites the body refuses to drop what it lacks.
	stdout, stderr, _ := desktop.env.RunCLI("push", "--rebuild")
	out := stdout + stderr
	if !strings.Contains(out, "refusing to rebuild") {
		t.Errorf("--rebuild was allowed to discard the other machine's frames; output:\n%s", out)
	}

	// The laptop's conversation must still be readable from the branch.
	reader := newMachine(t, bare, email, "reader")
	if _, stderr, err := reader.env.RunCLI("sync", "--self"); err != nil {
		t.Fatalf("reader sync --self: %v\n%s", err, stderr)
	}
	data, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "data.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader data.db: %v", err)
	}
	defer data.Close() //nolint:errcheck
	var n int
	if err := data.QueryRow(
		`SELECT count(*) FROM turns WHERE content LIKE '%laptop found the deadlock%'`,
	).Scan(&n); err != nil {
		t.Fatalf("query turns: %v", err)
	}
	if n == 0 {
		t.Error("the other machine's conversation is gone from the wire — nothing can restore it, since no data.db on any machine holds it")
	}
}

// TestPush_RebuildWorksWhenNothingIsLost is the other half. A guard that
// refuses every rebuild is not a guard, it is a removal — and --rebuild has a
// real job: re-encoding a branch whose wire bytes were written by a version
// with the frame-count bug. Rebuilding a branch this machine already contains
// loses nothing and must still work.
func TestPush_RebuildWorksWhenNothingIsLost(t *testing.T) {
	bare := newSharedRemote(t)
	solo := newMachine(t, bare, "solo-force@dev.example", "solo")
	solo.contribute(t, "s.jsonl",
		[]string{"opening", "the only machine here"}, "work")

	stdout, stderr, err := solo.env.RunCLI("push", "--rebuild")
	if err != nil {
		t.Fatalf("push --rebuild: %v\n%s", err, stderr)
	}
	out := stdout + stderr
	if strings.Contains(out, "refusing") {
		t.Errorf("--rebuild refused although the branch holds nothing this machine lacks; output:\n%s", out)
	}
}
