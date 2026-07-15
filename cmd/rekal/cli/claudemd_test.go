package cli

import (
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
		removeClaudeMDLine(dir)
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

		removeClaudeMDLine(dir)
		got = read(t, dir)
		if strings.Contains(got, rekalClaudeMDMarker) {
			t.Fatalf("marker line should be gone: %q", got)
		}
		if !strings.Contains(got, "Do not break this.") {
			t.Fatalf("user content lost on clean: %q", got)
		}
	})
}
