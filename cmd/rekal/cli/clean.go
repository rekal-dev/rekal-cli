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
  .rekal/            Data DB, index DB, and all local state
  post-commit hook   Only if it contains the rekal marker
  pre-push hook      Only if it contains the rekal marker
  agent skill        .claude/skills/rekal/ installed by 'rekal init'

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

// runClean removes .rekal/, Rekal hooks, and the installed skill. Idempotent.
func runClean(gitRoot string) error {
	rekalDir := RekalDir(gitRoot)
	if err := os.RemoveAll(rekalDir); err != nil {
		return fmt.Errorf("remove .rekal/: %w", err)
	}
	removeHook(filepath.Join(gitRoot, ".git", "hooks", "post-commit"))
	removeHook(filepath.Join(gitRoot, ".git", "hooks", "pre-push"))
	removeSkill(gitRoot)
	return nil
}

// removeSkill deletes the rekal-managed skill directory installed by init,
// then prunes .claude/skills/ and .claude/ if that left them empty — a
// user's own .claude content (settings, other skills) is never touched.
func removeSkill(gitRoot string) {
	skillDir := filepath.Join(gitRoot, ".claude", "skills", "rekal")
	_ = os.RemoveAll(skillDir)
	// os.Remove refuses non-empty directories, which is exactly the
	// behavior wanted here.
	_ = os.Remove(filepath.Join(gitRoot, ".claude", "skills"))
	_ = os.Remove(filepath.Join(gitRoot, ".claude"))
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
