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
	//
	// Compared by directory identity rather than by string: from a linked
	// worktree there is no caller spelling to preserve, so the answer is git's
	// own resolved path for the main checkout. On macOS that is /private/var/…
	// where t.TempDir() handed back /var/… — the same directory under another
	// name, which is all the store needs to be shared, and a string compare
	// fails there for no reason that matters.
	linked := t.TempDir() + "-wt"
	git(t, main, "worktree", "add", "-b", "feature", linked)
	got := MainWorktreeRoot(linked)
	gi, gErr := os.Stat(got)
	mi, mErr := os.Stat(main)
	if gErr != nil || mErr != nil || !os.SameFile(gi, mi) {
		t.Fatalf("linked worktree: MainWorktreeRoot(%q) = %q, want the main checkout %q", linked, got, main)
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

func TestTrackedBlobs_PathToSHA(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")

	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, dir, "add", path)
	}
	write("docs/a.md", "# A\n")
	write("docs/über notes.md", "# Über\n")
	git(t, dir, "commit", "-m", "add prose")

	blobs := TrackedBlobs(dir, "HEAD")
	if len(blobs) < 2 {
		t.Fatalf("TrackedBlobs = %v, want ≥2 paths", blobs)
	}
	for _, path := range []string{"docs/a.md", "docs/über notes.md"} {
		sha, ok := blobs[path]
		if !ok || sha == "" {
			t.Fatalf("missing blob for %q in %v", path, blobs)
		}
		want := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD:"+path))
		if sha != want {
			t.Fatalf("%s sha = %s, want %s", path, sha, want)
		}
	}

	// Rename keeps the blob SHA under the new path.
	oldSHA := blobs["docs/a.md"]
	git(t, dir, "mv", "docs/a.md", "docs/b.md")
	git(t, dir, "commit", "-m", "rename")
	after := TrackedBlobs(dir, "HEAD")
	if _, ok := after["docs/a.md"]; ok {
		t.Fatal("old path still in TrackedBlobs after rename")
	}
	if after["docs/b.md"] != oldSHA {
		t.Fatalf("renamed blob sha = %s, want %s", after["docs/b.md"], oldSHA)
	}
}

func TestBlobContents_Batch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	full := filepath.Join(dir, "docs/x.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("hello blob\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "docs/x.md")
	git(t, dir, "commit", "-m", "x")
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD:docs/x.md"))

	got := BlobContents(dir, []string{sha, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"})
	if string(got[sha]) != "hello blob\n" {
		t.Fatalf("BlobContents[%s] = %q", sha, got[sha])
	}
	if _, ok := got["deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"]; ok {
		t.Fatal("missing SHA should not appear in BlobContents")
	}
}

// TestMainWorktreeRoot_ThroughSymlink reproduces, on any platform, the failure
// that only shows up on macOS: there /var is a symlink to /private/var, so
// t.TempDir() hands back a path whose real location differs, and
// `git worktree list` reports the resolved one.
//
// The product promise is that the resolver is a no-op in the main checkout —
// that is what lets .rekal/ stay where it is with no migration. If reaching a
// repo through a symlinked path changes the answer, that promise is broken for
// real users too, not only for tests: the store would be looked for somewhere
// other than where it already sits.
func TestMainWorktreeRoot_ThroughSymlink(t *testing.T) {
	t.Parallel()
	real := t.TempDir()
	git(t, real, "init", "-b", "main")
	commit(t, real, "base")

	// A second path that reaches the same directory through a symlink, which is
	// exactly the shape of macOS's /var -> /private/var.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot create symlink here: %v", err)
	}

	if got := MainWorktreeRoot(link); got != filepath.Clean(link) {
		t.Errorf("MainWorktreeRoot(%q) = %q, want the caller's own path back — "+
			"the main checkout must resolve to itself however it was reached", link, got)
	}
}

// TestCommitMessages covers the lookup behind the commit-message facet signal.
//
// The whole message matters, not the subject: a subject is a label, the body is
// where the reasoning is written down, and that is the material worth indexing.
func TestCommitMessages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")

	body := "fix: stop the stampede\n\nA fixed delay retries in lockstep, so recovery\nre-floods the downstream. Exponential backoff with jitter instead."
	git(t, dir, "commit", "--allow-empty", "-m", body)
	first := git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "commit", "--allow-empty", "-m", "second subject only")
	second := git(t, dir, "rev-parse", "HEAD")

	got := CommitMessages(dir, []string{first, second})
	if len(got) != 2 {
		t.Fatalf("CommitMessages returned %d entries, want 2: %v", len(got), got)
	}
	if !strings.Contains(got[first], "Exponential backoff with jitter") {
		t.Errorf("the body was dropped — only the subject is indexed, which is the label rather than the reasoning:\n%q", got[first])
	}
	if !strings.Contains(got[first], "stop the stampede") {
		t.Errorf("the subject is missing from %q", got[first])
	}
	if got[second] != "second subject only" {
		t.Errorf("second = %q, want %q", got[second], "second subject only")
	}

	// A sha this clone does not have must not take the whole batch down.
	mixed := CommitMessages(dir, []string{first, "0000000000000000000000000000000000000000"})
	if _, ok := mixed[first]; ok && len(mixed) == 0 {
		t.Error("unreachable")
	}
	if len(CommitMessages(dir, nil)) != 0 {
		t.Error("empty input should return an empty map")
	}
}
