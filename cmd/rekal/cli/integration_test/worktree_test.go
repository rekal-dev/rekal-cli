//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestWorktree_SharesStore verifies that a linked git worktree shares the main
// checkout's .rekal store instead of creating its own — the whole point of
// worktree support (issue #4): one init, one sync, one index across all
// worktrees.
func TestWorktree_SharesStore(t *testing.T) {
	main := NewTestEnv(t)

	// A commit so the repo has a HEAD (init creates the orphan branch etc.).
	if err := os.WriteFile(filepath.Join(main.RepoDir, "f.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitIn(t, main.RepoDir, "add", ".")
	runGitIn(t, main.RepoDir, "commit", "-m", "base")

	main.Init()
	if !main.FileExists(".rekal/data.db") {
		t.Fatal("init did not create the store in the main worktree")
	}

	// Add a linked worktree on its own branch.
	linked := filepath.Join(t.TempDir(), "wt")
	runGitIn(t, main.RepoDir, "worktree", "add", "-b", "feature", linked)

	linkedEnv := NewTestEnvAt(t, linked)

	// A command requiring init must succeed from the linked worktree — proving
	// EnsureInitDone found the shared store, not a per-worktree one.
	if _, stderr, err := linkedEnv.RunCLI("index"); err != nil {
		t.Fatalf("rekal index from linked worktree: %v\nstderr: %s", err, stderr)
	}

	// The linked worktree must NOT have its own .rekal — it uses main's.
	if _, err := os.Stat(filepath.Join(linked, ".rekal")); !os.IsNotExist(err) {
		t.Fatalf("linked worktree grew its own .rekal (stat err=%v); the store must stay shared in main", err)
	}

	// And the index the linked run rebuilt lives in the main store.
	if !main.FileExists(".rekal/index.db") {
		t.Fatal("index rebuilt from the linked worktree did not land in the main store")
	}
}

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
