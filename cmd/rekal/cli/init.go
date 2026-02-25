package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

const rekalHookMarker = "# managed by rekal"

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Rekal in the current git repository",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
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

			fmt.Fprintln(cmd.OutOrStdout(), "Rekal initialized.")
			return nil
		},
	}

	return cmd
}

func ensureGitignore(gitRoot string) error {
	gitignorePath := filepath.Join(gitRoot, ".gitignore")
	entry := ".rekal/"

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

	// Append .rekal/ to .gitignore.
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline before entry if file doesn't end with one.
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
	if err := writeHook(postCommit, "#!/bin/sh\n"+rekalHookMarker+"\nrekal checkpoint\n"); err != nil {
		return fmt.Errorf("post-commit hook: %w", err)
	}

	prePush := filepath.Join(hooksDir, "pre-push")
	if err := writeHook(prePush, "#!/bin/sh\n"+rekalHookMarker+"\nrekal push\n"); err != nil {
		return fmt.Errorf("pre-push hook: %w", err)
	}

	return nil
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
// Otherwise, a new orphan branch is created with an empty initial commit.
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

	// Create new orphan branch with an empty initial commit.
	// We use git commands that don't affect the working tree or current branch.
	// Create an empty tree, commit it, and point the branch ref at it.
	emptyTree, err := exec.Command("git", "-C", gitRoot, "hash-object", "-t", "tree", "/dev/null").Output()
	if err != nil {
		return fmt.Errorf("create empty tree: %w", err)
	}
	treeHash := strings.TrimSpace(string(emptyTree))

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-m", "rekal: initialize checkpoint branch",
	).Output()
	if err != nil {
		return fmt.Errorf("create initial commit: %w", err)
	}
	commitHash := strings.TrimSpace(string(commitOut))

	return exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitHash).Run()
}
