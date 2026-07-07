package gitx

import (
	"os/exec"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes an empty commit and returns its SHA.
func commit(t *testing.T, dir, msg string) string {
	t.Helper()
	git(t, dir, "commit", "--allow-empty", "-m", msg)
	return git(t, dir, "rev-parse", "HEAD")
}

func TestIsAncestor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")

	base := commit(t, dir, "base")

	// Feature branch with its own commit.
	git(t, dir, "checkout", "-b", "feature")
	feat := commit(t, dir, "feature work")

	// main advances independently (feature not yet merged).
	git(t, dir, "checkout", "main")
	mainTip := commit(t, dir, "main advances")

	if !IsAncestor(dir, base, "main") {
		t.Error("base should be an ancestor of main")
	}
	if IsAncestor(dir, feat, "main") {
		t.Error("unmerged feature commit must NOT be an ancestor of main")
	}

	// Merge feature into main; now its commit is reachable.
	git(t, dir, "merge", "--no-ff", "-m", "merge feature", "feature")
	if !IsAncestor(dir, feat, "main") {
		t.Error("merged feature commit should be an ancestor of main")
	}
	if !IsAncestor(dir, mainTip, "main") {
		t.Error("main tip should be an ancestor of main")
	}

	// Guard rails.
	if IsAncestor(dir, "", "main") {
		t.Error("empty sha must not be an ancestor")
	}
	if IsAncestor(dir, base, "") {
		t.Error("empty ref must not report ancestry")
	}
}

func TestDefaultBranch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	commit(t, dir, "base")

	// No remote: falls back to local main.
	if got := DefaultBranch(dir); got != "main" {
		t.Fatalf("DefaultBranch = %q, want main", got)
	}
}

func TestDefaultBranch_EmptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	// No commits, no main/master ref resolvable.
	if got := DefaultBranch(dir); got != "" {
		t.Fatalf("DefaultBranch on empty repo = %q, want empty", got)
	}
}
