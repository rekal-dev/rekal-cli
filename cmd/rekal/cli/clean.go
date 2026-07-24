package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/skill"
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
  agent skill        .claude/skills/rekal/ (+ legacy rekal-* dirs)
  CLAUDE.md line     Only the marker-tagged sentence injected by 'rekal init'

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
	removeManagedLines(gitRoot)
	return nil
}

// removeManagedLines strips Rekal's marker-tagged line from every managed
// instructions file — CLAUDE.md plus the per-agent files init writes on
// detection (AGENTS.md, GEMINI.md, .github/copilot-instructions.md). Files left
// with only whitespace were ours and are deleted; a file with the user's own
// content keeps everything else. An emptied .github directory is pruned only if
// nothing else remains there.
func removeManagedLines(gitRoot string) {
	for _, rel := range []string{
		"CLAUDE.md",
		"AGENTS.md",
		"GEMINI.md",
		filepath.Join(".github", "copilot-instructions.md"),
		filepath.Join(".kiro", "steering", "rekal.md"),
	} {
		removeManagedLine(filepath.Join(gitRoot, rel))
	}
	// Prune directories we may have created solely for an agent file, when they
	// are left empty (os.Remove refuses a non-empty dir, so user content is
	// safe). .kiro/steering nests, so remove it before .kiro.
	_ = os.Remove(filepath.Join(gitRoot, ".github"))
	_ = os.Remove(filepath.Join(gitRoot, ".kiro", "steering"))
	_ = os.Remove(filepath.Join(gitRoot, ".kiro"))
}

// removeManagedLine deletes the marker-tagged line Rekal injected into the file
// at path. If nothing but whitespace remains the file is removed entirely (it
// was ours); a file with the user's own content keeps everything else. A file
// without the marker (or absent) is left untouched.
func removeManagedLine(path string) {
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), rekalClaudeMDMarker) {
		return
	}
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, rekalClaudeMDMarker) {
			continue
		}
		kept = append(kept, line)
	}
	out := strings.Join(kept, "\n")
	if strings.TrimSpace(out) == "" {
		_ = os.Remove(path)
		return
	}
	_ = os.WriteFile(path, []byte(strings.TrimRight(out, "\n")+"\n"), 0o644)
}

// removeSkill deletes every rekal-managed skill directory installed by init
// (current suite + legacy companion dirs), then prunes .claude/skills/ and
// .claude/ if that left them empty — a user's own .claude content (settings,
// other skills) is never touched.
func removeSkill(gitRoot string) {
	for _, s := range skill.All() {
		_ = os.RemoveAll(filepath.Join(gitRoot, ".claude", "skills", s.Name))
	}
	for _, name := range skill.LegacyNames {
		_ = os.RemoveAll(filepath.Join(gitRoot, ".claude", "skills", name))
	}
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
