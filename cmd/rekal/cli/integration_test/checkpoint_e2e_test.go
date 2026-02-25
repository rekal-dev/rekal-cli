//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/cli/cmd/rekal/cli/session"
)

const testSessionJSONL = `{"type":"summary","sessionId":"test-session-001","totalCost":0.05,"totalDuration":120}
{"type":"user","parentMessageId":"","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"fix the auth bug in login.go"}]},"timestamp":"2026-02-25T10:00:00Z","gitBranch":"main"}
{"type":"assistant","parentMessageId":"m1","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"Let me read the file first."},{"type":"tool_use","id":"tu-1","name":"Read","input":{"file_path":"login.go"}}]},"timestamp":"2026-02-25T10:00:30Z"}
{"type":"tool_result","parentMessageId":"m2","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu-1","content":"package main\n\nfunc login() {}"}]},"timestamp":"2026-02-25T10:00:31Z"}
{"type":"assistant","parentMessageId":"m3","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"I see the issue. Let me fix it."},{"type":"tool_use","id":"tu-2","name":"Edit","input":{"file_path":"login.go","old_string":"func login() {}","new_string":"func login() error { return nil }"}}]},"timestamp":"2026-02-25T10:01:00Z"}
{"type":"tool_result","parentMessageId":"m4","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu-2","content":"File edited successfully."}]},"timestamp":"2026-02-25T10:01:01Z"}
{"type":"assistant","parentMessageId":"m5","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"Fixed. The login function now returns an error."},{"type":"tool_use","id":"tu-3","name":"Bash","input":{"command":"go test ./..."}}]},"timestamp":"2026-02-25T10:01:30Z"}
{"type":"user","parentMessageId":"m7","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"looks good, thanks"}]},"timestamp":"2026-02-25T10:02:00Z"}
`

const testSessionJSONL2 = `{"type":"summary","sessionId":"test-session-002","totalCost":0.02,"totalDuration":60}
{"type":"user","parentMessageId":"","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"add error logging"}]},"timestamp":"2026-02-25T11:00:00Z","gitBranch":"feature/logging"}
{"type":"assistant","parentMessageId":"m1","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"I'll add error logging."},{"type":"tool_use","id":"tu-4","name":"Edit","input":{"file_path":"login.go","old_string":"return nil","new_string":"log.Println(\"ok\"); return nil"}}]},"timestamp":"2026-02-25T11:00:15Z"}
{"type":"user","parentMessageId":"m2","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"perfect"}]},"timestamp":"2026-02-25T11:00:30Z"}
`

// gitShow reads a file from a git ref. Returns nil if not found.
func gitShow(dir, ref, path string) []byte {
	out, err := exec.Command("git", "-C", dir, "show", ref+":"+path).Output()
	if err != nil {
		return nil
	}
	return out
}

// gitCommit stages all changes and creates a commit.
func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	if err := exec.Command("git", "-C", dir, "add", "-A").Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", msg).Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// writeSessionFile writes a .jsonl file to the Claude session directory for the repo.
// Returns a cleanup function to remove it.
func writeSessionFile(t *testing.T, repoDir, name, content string) func() {
	t.Helper()
	sessionDir := session.FindSessionDir(repoDir)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	path := filepath.Join(sessionDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	return func() {
		os.Remove(path)
		// Try to remove parent dir if empty.
		os.Remove(sessionDir)
	}
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestCheckpoint_E2E_FullPipeline(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Create initial commit so HEAD exists.
	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	branch := "rekal/test@rekal.dev"

	// Verify init created wire format files on orphan branch.
	bodyInit := gitShow(env.RepoDir, branch, "rekal.body")
	dictInit := gitShow(env.RepoDir, branch, "dict.bin")
	if bodyInit == nil {
		t.Fatal("rekal.body should exist on orphan branch after init")
	}
	if dictInit == nil {
		t.Fatal("dict.bin should exist on orphan branch after init")
	}
	if string(bodyInit[:7]) != "RKLBODY" {
		t.Errorf("body magic: got %q", bodyInit[:7])
	}
	if string(dictInit[:6]) != "RKDICT" {
		t.Errorf("dict magic: got %q", dictInit[:6])
	}
	if len(bodyInit) != 9 {
		t.Errorf("initial body should be 9 bytes (header only), got %d", len(bodyInit))
	}

	// --- First checkpoint ---

	// Place a session file.
	cleanup1 := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup1()

	// Make a commit so git diff HEAD~1 works.
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	// Run checkpoint.
	_, stderr, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stderr, "1 session(s) captured") {
		t.Errorf("expected '1 session(s) captured', got: %q", stderr)
	}

	// Verify DuckDB state.
	assertQueryContains(t, env, "SELECT count(*) as n FROM sessions", `"n":1`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM turns", `"n":5`)      // 2 user + 3 assistant
	assertQueryContains(t, env, "SELECT count(*) as n FROM tool_calls", `"n":3`) // Read, Edit, Bash
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoints", `"n":1`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoint_sessions", `"n":1`)

	// Verify wire format on orphan branch.
	body1 := gitShow(env.RepoDir, branch, "rekal.body")
	dict1 := gitShow(env.RepoDir, branch, "dict.bin")
	if body1 == nil || len(body1) <= 9 {
		t.Fatal("body should have frames after checkpoint")
	}
	if dict1 == nil || len(dict1) <= 12 {
		t.Fatal("dict should have entries after checkpoint")
	}

	// Scan frames — expect 3: session + checkpoint + meta.
	frames1, err := codec.ScanFrames(body1)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames1) != 3 {
		t.Fatalf("expected 3 frames after first checkpoint, got %d", len(frames1))
	}
	if frames1[0].Type != codec.FrameSession {
		t.Errorf("frame 0: expected session (0x01), got 0x%02x", frames1[0].Type)
	}
	if frames1[1].Type != codec.FrameCheckpoint {
		t.Errorf("frame 1: expected checkpoint (0x02), got 0x%02x", frames1[1].Type)
	}
	if frames1[2].Type != codec.FrameMeta {
		t.Errorf("frame 2: expected meta (0x03), got 0x%02x", frames1[2].Type)
	}

	// Decode session frame and verify data.
	dec, err := codec.NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	payload0 := codec.ExtractFramePayload(body1, frames1[0])
	sf, err := dec.DecodeSessionFrame(payload0)
	if err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if len(sf.Turns) != 5 {
		t.Errorf("session turns: got %d, want 5", len(sf.Turns))
	}
	if len(sf.ToolCalls) != 3 {
		t.Errorf("session tool_calls: got %d, want 3", len(sf.ToolCalls))
	}
	// Verify turn content survived roundtrip.
	if sf.Turns[0].Text != "fix the auth bug in login.go" {
		t.Errorf("turn 0 text: %q", sf.Turns[0].Text)
	}
	// Verify tool types.
	if sf.ToolCalls[0].Tool != codec.ToolRead {
		t.Errorf("tool 0: got %d, want Read (%d)", sf.ToolCalls[0].Tool, codec.ToolRead)
	}
	if sf.ToolCalls[1].Tool != codec.ToolEdit {
		t.Errorf("tool 1: got %d, want Edit (%d)", sf.ToolCalls[1].Tool, codec.ToolEdit)
	}
	if sf.ToolCalls[2].Tool != codec.ToolBash {
		t.Errorf("tool 2: got %d, want Bash (%d)", sf.ToolCalls[2].Tool, codec.ToolBash)
	}

	// Load dict and verify entries.
	loadedDict, err := codec.LoadDict(dict1)
	if err != nil {
		t.Fatalf("LoadDict: %v", err)
	}
	if loadedDict.Len(codec.NSSessions) != 1 {
		t.Errorf("dict sessions: %d", loadedDict.Len(codec.NSSessions))
	}
	if loadedDict.Len(codec.NSEmails) != 1 {
		t.Errorf("dict emails: %d", loadedDict.Len(codec.NSEmails))
	}

	// --- Idempotency: re-run checkpoint with same session ---

	_, stderr2, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint idempotent: %v", err)
	}
	// Should capture 0 new sessions (dedup by SHA-256).
	if strings.Contains(stderr2, "session(s) captured") {
		t.Error("idempotent checkpoint should not capture new sessions")
	}
	body1Again := gitShow(env.RepoDir, branch, "rekal.body")
	if sha256Hex(body1) != sha256Hex(body1Again) {
		t.Error("body should be unchanged after idempotent checkpoint")
	}

	// --- Second checkpoint: append-only property ---

	cleanup2 := writeSessionFile(t, env.RepoDir, "session2.jsonl", testSessionJSONL2)
	defer cleanup2()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { log.Println(\"ok\"); return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "add logging")

	_, stderr3, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint 2: %v (stderr: %s)", err, stderr3)
	}
	if !strings.Contains(stderr3, "1 session(s) captured") {
		t.Errorf("expected '1 session(s) captured', got: %q", stderr3)
	}

	body2 := gitShow(env.RepoDir, branch, "rekal.body")
	dict2 := gitShow(env.RepoDir, branch, "dict.bin")

	// Verify append-only: first N bytes of body2 must equal body1.
	if len(body2) <= len(body1) {
		t.Fatalf("body should grow: was %d, now %d", len(body1), len(body2))
	}
	prefix := body2[:len(body1)]
	if sha256Hex(prefix) != sha256Hex(body1) {
		t.Error("append-only violation: body prefix changed after second checkpoint")
	}

	// Should now have 6 frames.
	frames2, err := codec.ScanFrames(body2)
	if err != nil {
		t.Fatalf("ScanFrames 2: %v", err)
	}
	if len(frames2) != 6 {
		t.Fatalf("expected 6 frames after second checkpoint, got %d", len(frames2))
	}

	// Verify DuckDB totals.
	assertQueryContains(t, env, "SELECT count(*) as n FROM sessions", `"n":2`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoints", `"n":2`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoint_sessions", `"n":2`)

	// Verify dict grew.
	loadedDict2, err := codec.LoadDict(dict2)
	if err != nil {
		t.Fatalf("LoadDict 2: %v", err)
	}
	if loadedDict2.Len(codec.NSSessions) != 2 {
		t.Errorf("dict sessions after 2nd checkpoint: %d", loadedDict2.Len(codec.NSSessions))
	}

	// Decode the second session frame.
	payload3 := codec.ExtractFramePayload(body2, frames2[3])
	sf2, err := dec.DecodeSessionFrame(payload3)
	if err != nil {
		t.Fatalf("decode session 2: %v", err)
	}
	if len(sf2.Turns) != 3 {
		t.Errorf("session 2 turns: got %d, want 3", len(sf2.Turns))
	}
	if sf2.Turns[0].Text != "add error logging" {
		t.Errorf("session 2 turn 0: %q", sf2.Turns[0].Text)
	}

	t.Logf("E2E: body %d → %d bytes (delta %d), dict %d → %d bytes, 6 frames, 2 sessions, all data intact",
		len(body1), len(body2), len(body2)-len(body1), len(dict1), len(dict2))
}

func assertQueryContains(t *testing.T, env *TestEnv, sql, expected string) {
	t.Helper()
	stdout, _, err := env.RunCLI("query", sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	if !strings.Contains(stdout, expected) {
		t.Errorf("query %q: expected %q in output, got: %q", sql, expected, stdout)
	}
}
