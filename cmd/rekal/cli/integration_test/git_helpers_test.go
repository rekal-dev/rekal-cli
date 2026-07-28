//go:build integration

package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// sanitizedGitEnv is the environment every git command in these tests runs
// under.
//
// HOME and PATH are blanked deliberately. 'rekal init' installs a post-commit
// hook that resolves `rekal` through PATH and falls back to
// $HOME/.local/bin/rekal, so a git command run with the ambient environment
// would invoke whatever rekal build happens to be installed on the machine
// running the suite — capturing the session a second time, before the test's
// own explicit checkpoint call, and making results depend on the developer's
// workstation. The identity is fixed for the same reason results should not
// depend on a machine's git config.
func sanitizedGitEnv() []string {
	return append(os.Environ(),
		"HOME=/nonexistent", "PATH=/usr/bin:/bin",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
}

// gitRun runs a git command in dir and fails the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = sanitizedGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// gitOut runs a git command in dir and returns its trimmed stdout.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = sanitizedGitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

// currentBranch returns the repo's checked-out branch. Tests must not assume
// "main": NewTestEnv runs a bare `git init`, so the name comes from whatever
// init.defaultBranch the machine is configured with.
func currentBranch(t *testing.T, dir string) string {
	t.Helper()
	return gitOut(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// pushMain pushes the repo's current branch to origin, bypassing hooks.
func pushMain(t *testing.T, repoDir string) {
	t.Helper()
	gitRun(t, repoDir, "push", "--no-verify", "-q", "origin", currentBranch(t, repoDir))
}

// pullMain brings the repo's current branch up to date, as anyone returning to
// a second machine would before starting work again.
func pullMain(t *testing.T, repoDir string) {
	t.Helper()
	gitRun(t, repoDir, "pull", "--rebase", "-q", "origin", currentBranch(t, repoDir))
}
