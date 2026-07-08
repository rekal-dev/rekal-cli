package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/spf13/cobra"
)

// EnsureGitRoot resolves and returns the git repository root.
// Returns an error if the current directory is not inside a git repository.
func EnsureGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository; run from a git repo")
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureInitDone checks that Rekal has been initialized in the given git root.
// It verifies that .rekal/ exists and contains the expected database files.
func EnsureInitDone(gitRoot string) error {
	rekalDir := RekalDir(gitRoot)
	info, err := os.Stat(rekalDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("rekal not initialized; run 'rekal init' in a git repository")
	}
	dataDB := filepath.Join(rekalDir, "data.db")
	if _, err := os.Stat(dataDB); err != nil {
		return fmt.Errorf("rekal not initialized; run 'rekal init' in a git repository")
	}
	return nil
}

// RekalDir returns the path to the .rekal/ store, resolved to the repository's
// main worktree so all linked worktrees share one store (see
// gitx.MainWorktreeRoot). For a normal repo this is just <gitRoot>/.rekal.
func RekalDir(gitRoot string) string {
	return filepath.Join(gitx.MainWorktreeRoot(gitRoot), ".rekal")
}

// RequireInitializedRepo runs the two preconditions shared by every command
// except init and clean (docs/spec/preconditions.md): resolve the git root and
// verify Rekal is initialized. On failure it silences usage, prints the
// user-facing message, and returns a SilentError so the caller can just
// `return err`. This is the one central place the checks live, so the message
// and behavior stay identical across commands.
func RequireInitializedRepo(cmd *cobra.Command) (string, error) {
	cmd.SilenceUsage = true

	gitRoot, err := EnsureGitRoot()
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return "", NewSilentError(err)
	}
	if err := EnsureInitDone(gitRoot); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return "", NewSilentError(err)
	}
	return gitRoot, nil
}
