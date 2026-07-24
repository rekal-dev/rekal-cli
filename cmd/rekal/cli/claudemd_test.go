package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClaudeMDLine_InjectRefreshRemove covers the one-sentence CLAUDE.md
// surface end-to-end: create-when-missing, append-preserving-user-content,
// in-place refresh of the marker line, and clean removal that leaves the
// user's own content untouched (and no residue when the file was ours).
func TestClaudeMDLine_InjectRefreshRemove(t *testing.T) {
	t.Parallel()

	read := func(t *testing.T, dir string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
		if err != nil {
			t.Fatalf("read CLAUDE.md: %v", err)
		}
		return string(data)
	}

	t.Run("missing file is created and clean removes it", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := ensureClaudeMDLine(dir); err != nil {
			t.Fatalf("ensureClaudeMDLine: %v", err)
		}
		got := read(t, dir)
		if !strings.Contains(got, rekalClaudeMDMarker) || !strings.Contains(got, "`rekal` skill") {
			t.Fatalf("injected line missing: %q", got)
		}
		removeManagedLine(filepath.Join(dir, "CLAUDE.md"))
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
			t.Fatal("file we created should be removed entirely")
		}
	})

	t.Run("user content is preserved around the line", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		user := "# My project\n\nDo not break this.\n"
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(user), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ensureClaudeMDLine(dir); err != nil {
			t.Fatalf("ensureClaudeMDLine: %v", err)
		}
		got := read(t, dir)
		if !strings.HasPrefix(got, user) || !strings.Contains(got, rekalClaudeMDMarker) {
			t.Fatalf("append should preserve user content: %q", got)
		}

		// Refresh replaces the marker line in place — no duplicates.
		if err := ensureClaudeMDLine(dir); err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if strings.Count(read(t, dir), rekalClaudeMDMarker) != 1 {
			t.Fatal("refresh must not duplicate the line")
		}

		removeManagedLine(filepath.Join(dir, "CLAUDE.md"))
		got = read(t, dir)
		if strings.Contains(got, rekalClaudeMDMarker) {
			t.Fatalf("marker line should be gone: %q", got)
		}
		if !strings.Contains(got, "Do not break this.") {
			t.Fatalf("user content lost on clean: %q", got)
		}
	})
}

// TestAgentInstructions_DetectWriteRefreshRemove covers the per-agent
// instruction files: only detected agents' files are written (to the right
// path), a shared file is written once, refresh doesn't duplicate the line,
// user content is preserved, and clean removes ours.
func TestAgentInstructions_DetectWriteRefreshRemove(t *testing.T) {
	// Not parallel: uses t.Setenv to fake the machine's installed agents.
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Installed: codex (→ AGENTS.md) and gemini (→ GEMINI.md). Copilot absent.
	for _, d := range []string{".codex", ".gemini"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	repo := t.TempDir()
	agents := filepath.Join(repo, "AGENTS.md")
	gemini := filepath.Join(repo, "GEMINI.md")
	copilot := filepath.Join(repo, ".github", "copilot-instructions.md")

	// A user already has an AGENTS.md with their own content.
	userAgents := "# Agent rules\n\nAlways run tests.\n"
	if err := os.WriteFile(agents, []byte(userAgents), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	installAgentInstructions(repo, &buf)

	// Detected agents' files carry the marker; user content is preserved.
	for _, p := range []string{agents, gemini} {
		data, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(data), rekalClaudeMDMarker) {
			t.Fatalf("%s missing rekal line: %v / %q", p, err, string(data))
		}
	}
	if a, _ := os.ReadFile(agents); !strings.Contains(string(a), "Always run tests.") {
		t.Fatalf("AGENTS.md user content lost: %q", string(a))
	}
	// Copilot is not installed → its file must not exist.
	if _, err := os.Stat(copilot); !os.IsNotExist(err) {
		t.Fatal("copilot-instructions.md written though copilot not installed")
	}

	// Refresh is idempotent — the marker line is replaced, not duplicated.
	installAgentInstructions(repo, &buf)
	if a, _ := os.ReadFile(agents); strings.Count(string(a), rekalClaudeMDMarker) != 1 {
		t.Fatalf("refresh duplicated the AGENTS.md line: %q", string(a))
	}

	// Clean strips our line; the user's AGENTS.md keeps their content, and the
	// GEMINI.md we created is removed entirely.
	removeManagedLines(repo)
	a, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("AGENTS.md with user content should survive clean: %v", err)
	}
	if strings.Contains(string(a), rekalClaudeMDMarker) || !strings.Contains(string(a), "Always run tests.") {
		t.Fatalf("clean should drop our line and keep user content: %q", string(a))
	}
	if _, err := os.Stat(gemini); !os.IsNotExist(err) {
		t.Fatal("GEMINI.md we created should be removed on clean")
	}
}
