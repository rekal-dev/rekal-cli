package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectsRoot_ConfigDirOverride(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom/cfg")
	want := filepath.Join("/custom/cfg", "projects")
	if got := ProjectsRoot(); got != want {
		t.Fatalf("ProjectsRoot = %q, want %q", got, want)
	}
}

func TestEnumerateProjectDirs_MissingRootIsEmpty(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "does-not-exist"))
	dirs, err := EnumerateProjectDirs()
	if err != nil {
		t.Fatalf("EnumerateProjectDirs: %v", err)
	}
	if len(dirs) != 0 {
		t.Fatalf("expected no dirs for missing root, got %v", dirs)
	}
}

func TestEnumerateProjectDirs_ListsSubdirs(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	projects := filepath.Join(cfg, "projects")
	for _, name := range []string{"-home-a-repo", "-home-b-shell"} {
		if err := os.MkdirAll(filepath.Join(projects, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A stray file should be ignored (only directories are project roots).
	if err := os.WriteFile(filepath.Join(projects, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	dirs, err := EnumerateProjectDirs()
	if err != nil {
		t.Fatalf("EnumerateProjectDirs: %v", err)
	}
	if len(dirs) != 2 {
		t.Fatalf("expected 2 project dirs, got %d: %v", len(dirs), dirs)
	}
}

func TestProjectDirForRepo_MatchesSanitize(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	repo := "/home/frank/work/api"
	want := filepath.Join(cfg, "projects", SanitizeRepoPath(repo))
	if got := ProjectDirForRepo(repo); got != want {
		t.Fatalf("ProjectDirForRepo = %q, want %q", got, want)
	}
}
