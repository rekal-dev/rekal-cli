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
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/skill"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
	"github.com/spf13/cobra"
)

// skillVersionFile is written inside each installed skill dir
// (.claude/skills/<name>/.rekal-version) recording the binary Version that
// installed it. A later run compares it to detect a skill left behind by a
// binary upgrade — see maybeRefreshStaleSkill.
const skillVersionFile = ".rekal-version"

const rekalHookMarker = "# managed by rekal"

// rekalClaudeMDMarker tags the single line init injects into the repo's
// CLAUDE.md so refreshes replace it and clean removes it — no residue.
const rekalClaudeMDMarker = "<!-- managed by rekal -->"

// rekalClaudeMDLine is the one sentence of dev experience most users need:
// the skill carries the full routing policy; this line just makes it load.
const rekalClaudeMDLine = "Rekal memory is active here — before non-trivial work, use the `rekal` skill and route: grep the tree for present-tense code, `rekal` knowledge for present prose at HEAD, the ledger (`rekal` + gates) for past intent, the map for structure. " + rekalClaudeMDMarker

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
  agent skill        .claude/skills/rekal/ (tip + scripts/ + references/)
  CLAUDE.md line     One marker-tagged sentence pointing agents at the skill

If the remote already has data on your rekal branch, it is fetched and
imported into the local data DB automatically.

If Rekal is already initialized, 'init' leaves your data untouched and only
refreshes the version-managed skills and hooks — run it after upgrading the
binary to pick up new or changed skills.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			rekalDir := RekalDir(gitRoot)

			// Already initialized: don't rebuild the store, but do refresh the
			// version-managed assets (skills and hooks). These ship inside the
			// binary and are meant to track it, so `rekal init` after an upgrade
			// is how they get updated — without this, new/changed skills would
			// never reach an existing repo short of a full clean+init.
			if _, err := os.Stat(rekalDir); err == nil {
				if err := refreshManaged(gitRoot, cmd.ErrOrStderr()); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err)
					return NewSilentError(err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Rekal already initialized. Refreshed skills and hooks.")
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

			// One sentence into CLAUDE.md so the skill actually gets used —
			// the whole dev-experience surface for most users: init, done.
			if err := ensureClaudeMDLine(gitRoot); err != nil {
				return fmt.Errorf("update CLAUDE.md: %w", err)
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

// refreshManaged re-installs the assets that ship with the binary and are
// meant to track its version — the skill suite and the git hooks — plus the
// .claude gitignore entry. It touches no data: the store, orphan branch, and
// index are left exactly as they are. This is the upgrade path: run `rekal
// init` after updating the binary to pull new or changed skills into an
// existing repo without a clean+reimport.
func refreshManaged(gitRoot string, errOut io.Writer) error {
	if err := installHooks(gitRoot, errOut); err != nil {
		return fmt.Errorf("refresh hooks: %w", err)
	}
	if err := installSkill(gitRoot); err != nil {
		return fmt.Errorf("refresh skills: %w", err)
	}
	if err := ensureClaudeMDLine(gitRoot); err != nil {
		return fmt.Errorf("update CLAUDE.md: %w", err)
	}
	if err := ensureClaudeGitignore(gitRoot); err != nil {
		return fmt.Errorf("update .gitignore for .claude: %w", err)
	}
	return nil
}

// ensureClaudeMDLine injects (or refreshes) the one rekal sentence in the
// repo's CLAUDE.md. A line carrying the marker is replaced in place, so
// upgrades update the wording; otherwise the sentence is appended (creating
// the file when missing). The user's own content is never touched.
func ensureClaudeMDLine(gitRoot string) error {
	path := filepath.Join(gitRoot, "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(data)
	if strings.Contains(content, rekalClaudeMDMarker) {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, rekalClaudeMDMarker) {
				lines[i] = rekalClaudeMDLine
			}
		}
		return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	content += rekalClaudeMDLine + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// installSkill writes every Rekal-managed skill directory (SKILL.md plus
// scripts/ and references/) under .claude/skills/<name>/. Always overwrites —
// skills are managed by rekal and updated with each version. Also purges
// legacy companion skill dirs from older installs so the collapsed suite
// leaves no residue.
func installSkill(gitRoot string) error {
	for _, name := range skill.LegacyNames {
		_ = os.RemoveAll(filepath.Join(gitRoot, ".claude", "skills", name))
	}
	for _, s := range skill.All() {
		skillDir := filepath.Join(gitRoot, ".claude", "skills", s.Name)
		// Replace the whole dir so removed modules/scripts don't linger.
		_ = os.RemoveAll(skillDir)
		for rel, data := range s.Files {
			dst := filepath.Join(skillDir, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if skill.IsScript(rel) {
				mode = 0o755
			}
			if err := os.WriteFile(dst, data, mode); err != nil {
				return err
			}
		}
		// Pin the installing binary's version so a later run can detect a skill
		// left stale by an upgrade and refresh it. Written last, after the
		// RemoveAll+rewrite above.
		if err := os.WriteFile(filepath.Join(skillDir, skillVersionFile), []byte(Version+"\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// installedSkillVersion returns the binary Version pinned in the installed
// primary skill dir, or "" when the skill (or marker) is absent — i.e. never
// installed, or installed by a build predating the version pin.
func installedSkillVersion(gitRoot string) string {
	skills := skill.All()
	if len(skills) == 0 {
		return ""
	}
	path := filepath.Join(gitRoot, ".claude", "skills", skills[0].Name, skillVersionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// maybeRefreshStaleSkill re-installs the skill files when the version pinned in
// the installed skill differs from this binary's Version — so an upgraded
// binary's skill reaches the repo without a manual `rekal init`. It touches
// only the gitignored .claude/skills/ tree (never CLAUDE.md or hooks, which are
// tracked / side-effectful and stay the job of an explicit `rekal init`).
// Best-effort and gated on a version change, so the steady state is one small
// file read; skipped under bench so benchmarks never mutate the repo.
func maybeRefreshStaleSkill(gitRoot string, errOut io.Writer) {
	if session.BenchEnv() {
		return
	}
	installed := installedSkillVersion(gitRoot)
	if installed == Version {
		return
	}
	if err := installSkill(gitRoot); err != nil {
		fmt.Fprintf(errOut, "rekal: warning: skill refresh failed: %v\n", err)
		return
	}
	if installed == "" {
		fmt.Fprintf(errOut, "rekal: installed the %s skill (run `rekal init` to also refresh hooks)\n", Version)
	} else {
		fmt.Fprintf(errOut, "rekal: skill was behind (%s) — refreshed to %s (run `rekal init` to also refresh hooks)\n", installed, Version)
	}
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
