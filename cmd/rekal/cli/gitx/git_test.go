package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
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

// commitFile writes content to name and commits it, returning the commit sha.
func commitFile(t *testing.T, dir, name, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", name)
	git(t, dir, "commit", "-m", msg)
	return git(t, dir, "rev-parse", "HEAD")
}

func TestIsSquashMergedInto(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	commitFile(t, dir, "f.txt", "base\n", "base")

	// Feature branch with two real commits.
	git(t, dir, "checkout", "-b", "feature")
	mid := commitFile(t, dir, "f.txt", "base\none\n", "c1")
	tip := commitFile(t, dir, "f.txt", "base\none\ntwo\n", "c2")

	// Abandoned branch with its own change.
	git(t, dir, "checkout", "-b", "abandoned", "main")
	dead := commitFile(t, dir, "g.txt", "dead\n", "dead end")

	// Empty branch: commits but no tree change vs main.
	git(t, dir, "checkout", "-b", "empty", "main")
	git(t, dir, "commit", "--allow-empty", "-m", "no changes")
	emptyTip := git(t, dir, "rev-parse", "HEAD")

	// main advances independently, then squash-merges feature (GitHub style).
	git(t, dir, "checkout", "main")
	commitFile(t, dir, "h.txt", "mainwork\n", "main advances")
	git(t, dir, "merge", "--squash", "feature")
	git(t, dir, "commit", "-m", "feature (#1)")

	if !IsSquashMergedInto(dir, tip, "main") {
		t.Error("squash-merged branch tip must be detected")
	}
	if IsSquashMergedInto(dir, mid, "main") {
		t.Error("mid-branch commit is not the landed cumulative state — direct probe must fail (release happens via branch grouping)")
	}
	if IsSquashMergedInto(dir, dead, "main") {
		t.Error("abandoned branch must never be treated as squash-merged")
	}
	if IsSquashMergedInto(dir, emptyTip, "main") {
		t.Error("empty cumulative diff must fail closed")
	}
	if IsSquashMergedInto(dir, "", "main") || IsSquashMergedInto(dir, tip, "") {
		t.Error("empty args must fail closed")
	}
}

func TestMainWorktreeRoot(t *testing.T) {
	t.Parallel()
	main := t.TempDir()
	git(t, main, "init", "-b", "main")
	commit(t, main, "base")

	// In the main checkout, the resolver is a no-op (equals the checkout root).
	if got := MainWorktreeRoot(main); got != filepath.Clean(main) {
		t.Fatalf("main worktree: MainWorktreeRoot(%q) = %q, want the same", main, got)
	}

	// Add a linked worktree; from inside it, the resolver must point back at
	// the main checkout so the .rekal store is shared.
	linked := t.TempDir() + "-wt"
	git(t, main, "worktree", "add", "-b", "feature", linked)
	if got := MainWorktreeRoot(linked); got != filepath.Clean(main) {
		t.Fatalf("linked worktree: MainWorktreeRoot(%q) = %q, want main %q", linked, got, filepath.Clean(main))
	}
}

func TestMainWorktreeRoot_NonRepoFallsBack(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // not a git repo
	if got := MainWorktreeRoot(dir); got != dir {
		t.Fatalf("non-repo: MainWorktreeRoot(%q) = %q, want fallback to input", dir, got)
	}
}

func TestBranchTip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	sha := commit(t, dir, "base")

	if got := BranchTip(dir, "main"); got != sha {
		t.Fatalf("BranchTip(main) = %q, want %q", got, sha)
	}
	if got := BranchTip(dir, "gone"); got != "" {
		t.Fatalf("BranchTip(gone) = %q, want empty", got)
	}
	if got := BranchTip(dir, ""); got != "" {
		t.Fatalf("BranchTip(\"\") = %q, want empty", got)
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
