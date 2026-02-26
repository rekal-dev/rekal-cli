package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove Rekal setup from this repository (local only)",
		Long: `Remove Rekal setup from this repository. Local only — does not touch
the remote branch or .gitignore.

Removes:
  .rekal/           Data DB, index DB, and all local state
  post-commit hook   Only if it contains the rekal marker
  pre-push hook      Only if it contains the rekal marker

Run 'rekal init' to reinitialize after cleaning.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			if err := runClean(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Rekal cleaned. Run `rekal init` to reinitialize.")
			return nil
		},
	}
}

// runClean removes .rekal/ and Rekal hooks. Idempotent.
func runClean(gitRoot string) error {
	rekalDir := RekalDir(gitRoot)
	if err := os.RemoveAll(rekalDir); err != nil {
		return fmt.Errorf("remove .rekal/: %w", err)
	}
	removeHook(filepath.Join(gitRoot, ".git", "hooks", "post-commit"))
	removeHook(filepath.Join(gitRoot, ".git", "hooks", "pre-push"))
	return nil
}

// removeHook deletes a hook file only if it contains the rekal marker.
func removeHook(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if strings.Contains(string(data), rekalHookMarker) {
		_ = os.Remove(path)
	}
}
