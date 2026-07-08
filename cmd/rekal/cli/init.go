package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/skill"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
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
			if err := installHooks(gitRoot, cmd.ErrOrStderr()); err != nil {
				return fmt.Errorf("install hooks: %w", err)
			}

			// Create local orphan branch for checkpoint data.
			if err := transport.EnsureOrphanBranch(gitRoot); err != nil {
				return fmt.Errorf("create rekal branch: %w", err)
			}

			// Import existing data from orphan branch into DuckDB.
			branch := gitx.RekalBranchName()
			bodyData := gitx.ShowFile(gitRoot, branch, codec.BodyFilename)
			if len(bodyData) > 9 { // more than empty header
				importDB, err := db.OpenData(gitRoot)
				if err == nil {
					n, importErr := transport.ImportBranch(gitRoot, importDB, branch)
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

			// Pre-decompress nomic model so first query is fast.
			if err := nomic.WarmCache(); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: nomic cache warm failed: %v\n", err)
			}

			// Detect other AI agents and print integration hints.
			printAgentHints(cmd.ErrOrStderr(), gitRoot)

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

func installHooks(gitRoot string, w io.Writer) error {
	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	for _, h := range []struct {
		name       string
		subcommand string
	}{
		{"post-commit", "checkpoint"},
		{"pre-push", "push"},
	} {
		path := filepath.Join(hooksDir, h.name)
		installed, err := writeHook(path, hookScript(h.subcommand))
		if err != nil {
			return fmt.Errorf("%s hook: %w", h.name, err)
		}
		if !installed {
			// Say so instead of silently skipping — without the hook the
			// automatic capture/push never runs and nothing else hints why.
			fmt.Fprintf(w, "rekal: existing %s hook left untouched — add 'rekal %s' to it to enable automatic capture\n", h.name, h.subcommand)
		}
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

// writeHook installs a hook unless a foreign (non-rekal) hook already exists.
// Returns whether the hook was actually written.
func writeHook(path, content string) (bool, error) {
	// If a hook already exists and is not ours, leave it alone.
	existing, err := os.ReadFile(path)
	if err == nil && !strings.Contains(string(existing), rekalHookMarker) {
		return false, nil // not our hook; do not overwrite
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return false, err
	}
	return true, nil
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

// printAgentHints detects installed AI agents and prints integration hints.
// Writes a reference instructions file to .rekal/agent-instructions.md so
// users can copy the content into their AGENTS.md or GEMINI.md.
func printAgentHints(w io.Writer, gitRoot string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	var agents []string
	if _, err := os.Stat(filepath.Join(home, ".codex")); err == nil {
		agents = append(agents, "codex")
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini")); err == nil {
		agents = append(agents, "gemini")
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "opencode")); err == nil {
		agents = append(agents, "opencode")
	}

	if len(agents) == 0 {
		return
	}

	// Write reference instructions file.
	refPath := filepath.Join(RekalDir(gitRoot), "agent-instructions.md")
	_ = os.WriteFile(refPath, []byte(agentInstructionsContent), 0o644)

	for _, agent := range agents {
		switch agent {
		case "codex", "opencode":
			fmt.Fprintf(w, "rekal: %s detected — copy .rekal/agent-instructions.md into your AGENTS.md\n", agent)
		case "gemini":
			fmt.Fprintln(w, "rekal: gemini CLI detected — copy .rekal/agent-instructions.md into your GEMINI.md")
		}
	}
}

const agentInstructionsContent = `# Rekal — Session Memory

This repo uses Rekal to capture AI coding sessions. Use it to understand prior
context before modifying code.

## Quick Start

` + "```" + `bash
rekal "keyword"                        # search sessions
rekal --file src/auth/ "token refresh" # filter by file
rekal query --session <id>             # drill into a session
` + "```" + `

## When to Use

- Before modifying a file — check what prior sessions touched it
- When you need context about why code looks the way it does
- When working on files that were recently changed by AI agents

## Workflow

1. Search: ` + "`rekal \"keyword\"`" + `
2. Drill down: ` + "`rekal query --session <id> --offset N --limit 5`" + `
3. Full context: ` + "`rekal query --session <id> --full`" + `

Run ` + "`rekal --help`" + ` for all commands.
`
