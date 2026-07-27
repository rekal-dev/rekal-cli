package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRemoteDiverged_TransportFailureIsNotDivergence covers the misdiagnosis
// that sends a user to a destructive remedy.
//
// When the transport fails mid-push — a proxy answering 403, an expired
// credential, a branch-protection rule — git still prints "[rejected] ...
// (fetch first)", which reads exactly like a real divergence. Trusting that
// text told the user their history had diverged and pointed them at
// 'rekal push --force', which overwrites the shared branch with local data.
// The wire is the team's only copy of conversations already shared, so the
// advice is worse than the failure it was reacting to.
func TestRemoteDiverged_TransportFailureIsNotDivergence(t *testing.T) {
	t.Parallel()

	// The output git actually produced when a proxy refused the push.
	const proxy403 = `error: RPC failed; HTTP 403 curl 22 The requested URL returned error: 403
To http://127.0.0.1:41729/git/rekal-dev/rekal-cli
 ! [rejected]          rekal/x@y.z -> rekal/x@y.z (fetch first)
error: failed to push some refs`

	// The text alone still looks like a rejection — that is the trap.
	if !isNonFastForward(proxy403) {
		t.Fatal("setup: this output does read as a rejection; the guard has to come from the refs, not the text")
	}

	// No reachable remote, so the fetch fails: report the transport error, and
	// never claim a divergence that was never observed.
	dir := t.TempDir()
	run(t, dir, "init", "-b", "main")
	run(t, dir, "commit", "--allow-empty", "-m", "base")
	run(t, dir, "remote", "add", "origin", filepath.Join(dir, "nonexistent.git"))

	if remoteDiverged(dir, "main") {
		t.Error("remoteDiverged = true with an unreachable remote — a connection that never happened cannot prove history diverged")
	}
}

// TestRemoteDiverged_RealDivergenceIsReported is the other half: when the
// remote genuinely carries a commit local does not, say so, because there the
// force path is the legitimate answer.
func TestRemoteDiverged_RealDivergenceIsReported(t *testing.T) {
	t.Parallel()

	remote := t.TempDir()
	run(t, remote, "init", "--bare", "-b", "main")

	local := t.TempDir()
	run(t, local, "init", "-b", "main")
	run(t, local, "remote", "add", "origin", remote)
	run(t, local, "commit", "--allow-empty", "-m", "shared")
	run(t, local, "push", "origin", "main")

	// Another clone adds a commit the local branch will never contain.
	other := t.TempDir()
	run(t, other, "clone", remote, other)
	run(t, other, "commit", "--allow-empty", "-m", "theirs")
	run(t, other, "push", "origin", "main")

	// Local rewrites its own history instead of pulling.
	run(t, local, "commit", "--allow-empty", "--amend", "-m", "mine")

	if !remoteDiverged(local, "main") {
		t.Error("remoteDiverged = false, want true — the remote holds a commit local does not, which is the case --force exists for")
	}
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
