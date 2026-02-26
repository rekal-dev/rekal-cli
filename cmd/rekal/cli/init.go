package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/cli/cmd/rekal/cli/skill"
	"github.com/spf13/cobra"
)

const rekalHookMarker = "# managed by rekal"

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Rekal in the current git repository",
		Long: `Initialize Rekal in the current git repository.

Creates:
  .rekal/           Local directory (gitignored) with data.db and index.db
  post-commit hook   Runs 'rekal checkpoint' after each commit
  pre-push hook      Runs 'rekal push' before each push
  orphan branch      rekal/<email> for wire format storage
  agent skill        .claude/skills/rekal/SKILL.md for Claude Code

If the remote already has data on your rekal branch, it is fetched and
imported into the local data DB automatically.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			if err := ensureClaudeInstalled(); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "rekal requires Claude Code, which was not detected on this system.")
				fmt.Fprintln(cmd.ErrOrStderr(), "For the beta release, only Claude Code is supported. Other coding agents will be supported in a future release.")
				fmt.Fprintln(cmd.ErrOrStderr(), "Install Claude Code from https://docs.anthropic.com/en/docs/claude-code then run 'rekal init' again.")
				return NewSilentError(err)
			}

			rekalDir := RekalDir(gitRoot)

			if _, err := os.Stat(rekalDir); err == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Rekal is already initialized. Run 'rekal clean' first to reinitialize.")
				return nil
			}

			// Create .rekal/ directory.
			if err := os.MkdirAll(rekalDir, 0o755); err != nil {
				return fmt.Errorf("create .rekal/: %w", err)
			}

			// Create data DB with schema.
			dataDB, err := db.OpenData(gitRoot)
			if err != nil {
				return fmt.Errorf("create data DB: %w", err)
			}
			if err := db.InitDataSchema(dataDB); err != nil {
				dataDB.Close()
				return fmt.Errorf("init data schema: %w", err)
			}
			dataDB.Close()

			// Create index DB with schema.
			indexDB, err := db.OpenIndex(gitRoot)
			if err != nil {
				return fmt.Errorf("create index DB: %w", err)
			}
			if err := db.InitIndexSchema(indexDB); err != nil {
				indexDB.Close()
				return fmt.Errorf("init index schema: %w", err)
			}
			indexDB.Close()

			// Ensure .rekal/ is in .gitignore.
			if err := ensureGitignore(gitRoot); err != nil {
				return fmt.Errorf("update .gitignore: %w", err)
			}

			// Install hook stubs.
			if err := installHooks(gitRoot); err != nil {
				return fmt.Errorf("install hooks: %w", err)
			}

			// Create local orphan branch for checkpoint data.
			if err := ensureOrphanBranch(gitRoot); err != nil {
				return fmt.Errorf("create rekal branch: %w", err)
			}

			// Import existing data from orphan branch into DuckDB.
			branch := rekalBranchName()
			bodyData := gitShowFile(gitRoot, branch, "rekal.body")
			if len(bodyData) > 9 { // more than empty header
				importDB, err := db.OpenData(gitRoot)
				if err == nil {
					n, importErr := importBranch(gitRoot, importDB, branch)
					importDB.Close()
					if importErr != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "rekal: import error: %v\n", importErr)
					} else if n > 0 {
						fmt.Fprintf(cmd.ErrOrStderr(), "rekal: imported %d session(s) from remote\n", n)
					}
				}
			}

			// Install Claude Code skill.
			if err := installSkill(gitRoot); err != nil {
				return fmt.Errorf("install skill: %w", err)
			}

			// Gitignore .claude/ or just .claude/skills/ depending on whether
			// the user already has a .claude directory (settings, CLAUDE.md, etc.).
			if err := ensureClaudeGitignore(gitRoot); err != nil {
				return fmt.Errorf("update .gitignore for .claude: %w", err)
			}

			// Run initial checkpoint to capture any existing sessions.
			if err := doCheckpoint(gitRoot, cmd.ErrOrStderr()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: initial checkpoint failed: %v\n", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Rekal initialized.")
			return nil
		},
	}

	return cmd
}

func ensureGitignore(gitRoot string) error {
	return appendGitignoreEntry(gitRoot, ".rekal/")
}

// appendGitignoreEntry adds entry to .gitignore if not already present.
func appendGitignoreEntry(gitRoot, entry string) error {
	gitignorePath := filepath.Join(gitRoot, ".gitignore")

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return nil // already present
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

func installHooks(gitRoot string) error {
	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	postCommit := filepath.Join(hooksDir, "post-commit")
	if err := writeHook(postCommit, hookScript("checkpoint")); err != nil {
		return fmt.Errorf("post-commit hook: %w", err)
	}

	prePush := filepath.Join(hooksDir, "pre-push")
	if err := writeHook(prePush, hookScript("push")); err != nil {
		return fmt.Errorf("pre-push hook: %w", err)
	}

	return nil
}

// hookScript generates a shell hook that resolves the rekal binary at runtime.
// Checks PATH first, then falls back to ~/.local/bin/rekal (the default install location).
func hookScript(subcommand string) string {
	return `#!/bin/sh
` + rekalHookMarker + `
if command -v rekal >/dev/null 2>&1; then
  rekal ` + subcommand + `
elif [ -x "$HOME/.local/bin/rekal" ]; then
  "$HOME/.local/bin/rekal" ` + subcommand + `
fi
`
}

func writeHook(path, content string) error {
	// If a hook already exists and is not ours, leave it alone.
	existing, err := os.ReadFile(path)
	if err == nil && !strings.Contains(string(existing), rekalHookMarker) {
		return nil // not our hook; do not overwrite
	}
	return os.WriteFile(path, []byte(content), 0o755)
}

// rekalBranchName returns the orphan branch name for the current user.
// Format: rekal/<user_email>
func rekalBranchName() string {
	email := strings.TrimSpace(gitConfigValue("user.email"))
	if email == "" {
		email = "local"
	}
	return "rekal/" + email
}

// gitConfigValue reads a git config value.
func gitConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ensureOrphanBranch creates or fetches the local rekal orphan branch.
// If the branch exists locally, it's left as-is.
// If it exists on the remote, it's fetched.
// Otherwise, a new orphan branch is created with empty rekal.body and dict.bin.
func ensureOrphanBranch(gitRoot string) error {
	branch := rekalBranchName()

	// Check if local branch already exists.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err == nil {
		return nil // already exists locally
	}

	// Check if remote branch exists and fetch it.
	remote := "origin"
	remoteBranch := remote + "/" + branch
	// Fetch the specific branch (ignore errors — remote may not exist or branch may not exist).
	_ = exec.Command("git", "-C", gitRoot, "fetch", remote, branch).Run()

	// If remote branch now exists locally as a remote-tracking branch, create local from it.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", remoteBranch).Run(); err == nil {
		return exec.Command("git", "-C", gitRoot, "branch", branch, remoteBranch).Run()
	}

	// Create new orphan branch with initial wire format files.
	bodyData := codec.NewBody()
	dictData := codec.NewDict().Encode()

	bodyHash, err := gitHashObject(gitRoot, bodyData)
	if err != nil {
		return fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitHashObject(gitRoot, dictData)
	if err != nil {
		return fmt.Errorf("hash dict.bin: %w", err)
	}

	treeEntry := fmt.Sprintf("100644 blob %s\tdict.bin\n100644 blob %s\trekal.body\n", dictHash, bodyHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-m", "rekal: initialize checkpoint branch",
	).Output()
	if err != nil {
		return fmt.Errorf("create initial commit: %w", err)
	}
	commitHash := strings.TrimSpace(string(commitOut))

	return exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitHash).Run()
}

// gitHashObject writes data to the git object store and returns its hash.
func gitHashObject(gitRoot string, data []byte) (string, error) {
	cmd := exec.Command("git", "-C", gitRoot, "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// installSkill writes the Rekal skill to .claude/skills/rekal/SKILL.md.
// Always overwrites — the skill is managed by rekal and updated with each version.
func installSkill(gitRoot string) error {
	skillDir := filepath.Join(gitRoot, ".claude", "skills", "rekal")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill.RekalSkill), 0o644)
}

// ensureClaudeGitignore adds the appropriate .claude gitignore entry.
// If .claude/ already exists (user has settings, CLAUDE.md, etc.), only ignore
// .claude/skills/ so the skill doesn't get committed. Otherwise ignore the
// entire .claude/ directory.
func ensureClaudeGitignore(gitRoot string) error {
	claudeDir := filepath.Join(gitRoot, ".claude")

	// Determine what to ignore: if .claude/ existed before we created skills/,
	// the user has their own .claude content — only ignore the skills subfolder.
	// We check for entries other than "skills" in .claude/.
	entry := ".claude/"
	entries, err := os.ReadDir(claudeDir)
	if err == nil {
		for _, e := range entries {
			if e.Name() != "skills" {
				entry = ".claude/skills/"
				break
			}
		}
	}

	return appendGitignoreEntry(gitRoot, entry)
}

// ensureClaudeInstalled checks that Claude Code is installed.
// Looks for the "claude" binary on PATH or the ~/.claude/ config directory.
func ensureClaudeInstalled() error {
	// Check PATH first.
	if _, err := exec.LookPath("claude"); err == nil {
		return nil
	}

	// Fall back to config directory.
	home, err := os.UserHomeDir()
	if err == nil {
		if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
			return nil
		}
	}

	return fmt.Errorf("claude code not detected")
}
