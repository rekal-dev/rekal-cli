package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push Rekal data to the remote branch",
		Long: `Export new checkpoints to the wire format and push to the remote orphan branch.

Only YOUR unexported checkpoints are pushed — team data imported via 'rekal sync'
is never re-exported. Each user pushes to their own branch (rekal/<email>).

Checkpoints contain sessions (conversation turns, tool calls) and file change
metadata anchored to git commits. They are encoded into a compact binary wire
format (rekal.body + dict.bin) using zstd compression and string interning —
a 2-10 MB session compresses to ~300 bytes on the wire.

Use --force to overwrite the remote branch when it has diverged from local
(e.g. after a rebuild or conflict).

Normally runs automatically via the pre-push git hook installed by 'rekal init'.
You do not need to run this manually.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}
			if err := EnsureInitDone(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			return doPush(gitRoot, cmd.ErrOrStderr(), force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force push (overwrite remote with local data)")
	return cmd
}

// doPush pushes Rekal data to the remote orphan branch.
// Extracted so sync can call it without a cobra.Command.
func doPush(gitRoot string, w io.Writer, force bool) error {
	branch := rekalBranchName()

	// Check if local branch exists — if not, nothing to push.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err != nil {
		fmt.Fprintln(w, "rekal: no data to push (run 'rekal checkpoint' first)")
		return nil
	}

	// Check if remote is configured.
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		fmt.Fprintln(w, "rekal: no remote 'origin' configured — skipping push")
		return nil
	}

	// Export unexported checkpoints from DuckDB → wire format → orphan branch.
	body, dict, err := exportNewFrames(gitRoot)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	if body != nil {
		if _, err := commitWireFormat(gitRoot, body, dict); err != nil {
			return fmt.Errorf("commit to rekal branch: %w", err)
		}
	} else {
		fmt.Fprintln(w, "rekal: no new checkpoints to export")
	}

	// Compare local SHA vs remote tracking SHA — skip if identical.
	localSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return nil
	}
	remoteSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", "origin/"+branch).Output()
	if err == nil && strings.TrimSpace(string(localSHA)) == strings.TrimSpace(string(remoteSHA)) {
		fmt.Fprintln(w, "rekal: already up to date")
		return nil
	}

	if force {
		forceCmd := exec.Command("git", "-C", gitRoot, "push", "--no-verify", "--force", "origin", branch)
		forceCmd.Stdin = nil
		if output, err := forceCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(w, "rekal: force push failed: %s\n", strings.TrimSpace(string(output)))
			return nil
		}
		fmt.Fprintf(w, "rekal: force pushed to origin/%s\n", branch)
		return nil
	}

	// Push with --no-verify to prevent recursive pre-push hook.
	pushCmd := exec.Command("git", "-C", gitRoot, "push", "--no-verify", "origin", branch)
	pushCmd.Stdin = nil // disconnect stdin so git doesn't hang in hook context
	output, err := pushCmd.CombinedOutput()
	if err != nil {
		if isNonFastForward(string(output)) {
			fmt.Fprintf(w, "rekal: push rejected (non-fast-forward) for origin/%s\n", branch)
			fmt.Fprintln(w, "rekal: your remote branch has diverged from local — review and run 'rekal push --force' to overwrite remote with local data")
			return nil
		}
		fmt.Fprintf(w, "rekal: push failed: %s\n", strings.TrimSpace(string(output)))
		return nil
	}

	fmt.Fprintf(w, "rekal: pushed to origin/%s\n", branch)
	return nil
}

// isNonFastForward checks if git push output indicates a non-fast-forward rejection.
func isNonFastForward(output string) bool {
	return strings.Contains(output, "non-fast-forward") ||
		strings.Contains(output, "[rejected]") ||
		strings.Contains(output, "fetch first")
}
