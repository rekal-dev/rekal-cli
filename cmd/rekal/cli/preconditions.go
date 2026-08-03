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

// rejectExtraArgs is the Args validator for commands that take no
// positional arguments of their own (log, version, sync, push, embed,
// index, checkpoint, clean, init). A bare invocation (`rekal log`) is
// deliberately still valid — that's ordinary subcommand dispatch, same as
// any other CLI. But trailing words are the signature of a natural-language
// query whose first word happened to match a command name: cobra resolves
// `rekal log recent commits about the ledger` to this command and hands
// RunE the rest, which a `_ []string` RunE silently discards — the caller
// gets the bare command's unrelated output with no error at all. This turns
// that into a loud, actionable one instead: `rekal -- <query>` is the
// existing (undocumented until now) escape hatch that forces root-level
// recall past the name collision, because `--` stops cobra from resolving
// the next word as a subcommand.
func rejectExtraArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	full := cmd.Name() + " " + strings.Join(args, " ")
	return fmt.Errorf(
		"rekal %s: takes no arguments (got %d)\n"+
			"rekal: if you meant to search for %q, the leading word collided with the %q command — run: rekal -- %s",
		cmd.Name(), len(args), full, cmd.Name(), full)
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
