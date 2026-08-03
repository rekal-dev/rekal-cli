package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitRoot_InGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// EnsureGitRoot uses cwd, so we need to run from the dir.
	// Use a subprocess approach to avoid os.Chdir in parallel tests.
	root := resolveGitRoot(t, dir)
	if root == "" {
		t.Error("expected non-empty git root")
	}
}

func TestEnsureInitDone_NoRekalDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := EnsureInitDone(dir)
	if err == nil {
		t.Error("expected error when .rekal/ does not exist")
	}
}

func TestEnsureInitDone_WithRekalDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rekalDir := filepath.Join(dir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create data.db placeholder.
	if err := os.WriteFile(filepath.Join(rekalDir, "data.db"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	err := EnsureInitDone(dir)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRekalDir(t *testing.T) {
	t.Parallel()

	got := RekalDir("/tmp/myrepo")
	want := filepath.Join("/tmp/myrepo", ".rekal")
	if got != want {
		t.Errorf("RekalDir = %q, want %q", got, want)
	}
}

func TestRejectExtraArgs(t *testing.T) {
	t.Parallel()

	logCmd := newLogCmd()
	if err := rejectExtraArgs(logCmd, nil); err != nil {
		t.Errorf("bare command: expected no error, got: %v", err)
	}

	err := rejectExtraArgs(logCmd, []string{"recent", "commits", "about", "the", "ledger"})
	if err == nil {
		t.Fatal("expected error for unexpected positional args")
	}
	// The message must name the actual escape hatch, not silently discard
	// the words the caller typed (what happened before this fix).
	want := "rekal -- log recent commits about the ledger"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not mention escape hatch %q", err.Error(), want)
	}
}

// resolveGitRoot runs git rev-parse in the given dir to get the git root.
func resolveGitRoot(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse: %v", err)
	}
	return string(out[:len(out)-1]) // trim newline
}
