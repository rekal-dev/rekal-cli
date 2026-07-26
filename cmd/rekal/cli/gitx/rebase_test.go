package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeRepoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestRebaseInProgress verifies the signal capture relies on to skip replayed
// commits. GIT_REFLOG_ACTION is deliberately not used: it is empty in the
// post-commit environment during a rebase, so only git's own state directories
// are a reliable indicator.
func TestRebaseInProgress(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	git(t, dir, "init", "-q", "-b", "main")
	writeRepoFile(t, dir, "base.txt", "base\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")

	if RebaseInProgress(dir) {
		t.Fatal("clean repo reported as mid-rebase")
	}

	// Two branches that touch the same file, so the rebase stops on conflict
	// and leaves the rebase state directory in place for inspection.
	git(t, dir, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "shared.txt", "feature side\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "feature edit")

	git(t, dir, "checkout", "-q", "main")
	writeRepoFile(t, dir, "shared.txt", "main side\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "main edit")

	git(t, dir, "checkout", "-q", "feature")
	// Expected to fail with a conflict; that is the point.
	_ = exec.Command("git", "-C", dir, "rebase", "main").Run()

	if !RebaseInProgress(dir) {
		t.Error("mid-rebase repo not detected — capture would run on every replayed commit")
	}

	git(t, dir, "rebase", "--abort")
	if RebaseInProgress(dir) {
		t.Error("still reported as mid-rebase after abort")
	}
}

// TestRebaseInProgress_NotAGitRepo ensures the probe fails closed: an unreadable
// git dir must not be mistaken for an in-progress rebase, or capture would
// silently stop working.
func TestRebaseInProgress_NotAGitRepo(t *testing.T) {
	t.Parallel()
	if RebaseInProgress(t.TempDir()) {
		t.Error("non-repo reported as mid-rebase")
	}
}
