//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestCheckpoint_CopilotDiscoveryCacheInvalidatesFile(t *testing.T) {
	t.Setenv("COPILOT_HOME", t.TempDir())
	env := NewTestEnv(t)
	env.Init()
	gitCommit(t, env.RepoDir, "initial")
	write := func(id, cwd string) string {
		t.Helper()
		dir := filepath.Join(os.Getenv("COPILOT_HOME"), "session-state", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "events.jsonl")
		body := fmt.Sprintf("{\"type\":\"session.start\",\"data\":{\"sessionId\":%q,\"context\":{\"cwd\":%q}}}\n{\"type\":\"user.message\",\"data\":{\"content\":\"keep this evidence\"}}\n", id, cwd)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for i := 0; i < 24; i++ {
		write(fmt.Sprintf("other-%02d", i), "/another/repo")
	}
	write("matching", env.RepoDir)
	checkpoint := func() {
		t.Helper()
		if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
			t.Fatalf("checkpoint: %v: %s", err, stderr)
		}
	}
	checkpoint()
	checkpoint() // separate command; reload persistent positive and negative entries
	d := openStore(t, env.RepoDir)
	if got := countRows(t, d, "SELECT count(*) FROM session_discovery_cache WHERE adapter='copilot'"); got != 25 {
		t.Fatalf("cached inventory=%d, want 25", got)
	}
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 1 {
		t.Fatalf("sessions=%d, want 1", got)
	}
	d.Close() //nolint:errcheck

	path := filepath.Join(os.Getenv("COPILOT_HOME"), "session-state", "other-00", "events.jsonl")
	parent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	write("other-00", env.RepoDir) // existing file; directory mtime stays unchanged
	after, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if !parent.ModTime().Equal(after.ModTime()) {
		t.Fatal("fixture changed parent directory mtime")
	}
	checkpoint()
	d = openStore(t, env.RepoDir)
	defer d.Close() //nolint:errcheck
	if got := countRows(t, d, "SELECT count(*) FROM sessions"); got != 2 {
		t.Fatalf("changed previously nonmatching file was missed: sessions=%d", got)
	}
}

func TestCheckpoint_CodexCompressionRetainsAppendIdentity(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	env := NewTestEnv(t)
	env.Init()
	gitCommit(t, env.RepoDir, "initial")
	dir := filepath.Join(os.Getenv("CODEX_HOME"), "sessions", "2026", "11", "06")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout.jsonl")
	body := fmt.Sprintf("{\"type\":\"session_meta\",\"payload\":{\"id\":\"compress-resume\",\"cwd\":%q}}\n", env.RepoDir) +
		"{\"type\":\"event_msg\",\"payload\":{\"type\":\"user_message\",\"message\":\"first question\"}}\n"
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	checkpoint := func() {
		t.Helper()
		if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
			t.Fatalf("checkpoint: %v: %s", err, stderr)
		}
	}
	write(path, []byte(body))
	checkpoint()
	w, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	write(path+".zst", w.EncodeAll([]byte(body), nil))
	w.Close() //nolint:errcheck
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	checkpoint() // compressing an already captured rollout creates no new ledger rows
	body += "{\"type\":\"event_msg\",\"payload\":{\"type\":\"agent_message\",\"message\":\"answer after resume\"}}\n"
	write(path, []byte(body))
	if err := os.Remove(path + ".zst"); err != nil {
		t.Fatal(err)
	}
	checkpoint() // Codex materializes on resume; append to the same session
	d := openStore(t, env.RepoDir)
	defer d.Close() //nolint:errcheck
	for query, want := range map[string]int{
		"SELECT count(*) FROM sessions":         1,
		"SELECT count(*) FROM turns":            2,
		"SELECT count(*) FROM checkpoints":      2,
		"SELECT count(*) FROM checkpoint_state": 1,
	} {
		if got := countRows(t, d, query); got != want {
			t.Errorf("%s = %d, want %d", query, got, want)
		}
	}
}
