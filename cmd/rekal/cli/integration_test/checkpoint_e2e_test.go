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

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
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

// gitCommit stages all changes and creates a commit. The commit runs with a
// sanitized environment so the post-commit hook installed by 'rekal init'
// (which resolves 'rekal' via PATH, falling back to $HOME/.local/bin/rekal)
// can't find and invoke whatever rekal build happens to be installed on the
// machine running the test, double-capturing the session before the test's
// own explicit checkpoint call.
func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	env := append(os.Environ(), "HOME=/nonexistent", "PATH=/usr/bin:/bin")
	addCmd := exec.Command("git", "-C", dir, "add", "-A")
	addCmd.Env = env
	if err := addCmd.Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}
	commitCmd := exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", msg)
	commitCmd.Env = env
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// writeSessionFile writes a .jsonl file to the Claude session directory for the repo.
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

	// --- First checkpoint (DuckDB only, no wire format) ---

	cleanup1 := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup1()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

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

	// Checkpoint should NOT have written to orphan branch (that's push's job now).
	bodyAfterCp := gitShow(env.RepoDir, branch, "rekal.body")
	if len(bodyAfterCp) != 9 {
		t.Errorf("body should still be header-only after checkpoint (no wire format), got %d bytes", len(bodyAfterCp))
	}

	// Checkpoint should be unexported.
	assertQueryContains(t, env, "SELECT exported FROM checkpoints", `"exported":false`)

	// --- Idempotency: re-run checkpoint with same session ---

	_, stderr2, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint idempotent: %v", err)
	}
	if strings.Contains(stderr2, "session(s) captured") {
		t.Error("idempotent checkpoint should not capture new sessions")
	}

	// --- Re-capture after the conversation GREW ---
	//
	// The unchanged case above is the trivial half: the size+hash fast-skip
	// short-circuits before any dedup logic runs. The half that mattered — and
	// that went unasserted while a duplicate-session bug lived here — is a
	// transcript that gained turns between two commits, which is what happens
	// whenever a session spans more than one commit. It must extend the existing
	// session, not store the conversation a second time.
	grown := testSessionJSONL +
		`{"type":"user","parentMessageId":"m8","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"one more thing before we ship"}]},"timestamp":"2026-02-25T10:03:00Z"}` + "\n"
	writeSessionFile(t, env.RepoDir, "session1.jsonl", grown)
	gitCommit(t, env.RepoDir, "second commit, same conversation")

	if _, _, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint after growth: %v", err)
	}
	assertQueryContains(t, env,
		"SELECT count(*) AS n FROM sessions", `"n":1`)
	assertQueryContains(t, env,
		"SELECT content FROM turns ORDER BY turn_index DESC LIMIT 1", "one more thing before we ship")

	// --- Second checkpoint ---

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

	// Two conversations, three commits: session1 was captured, grew, and was
	// captured again. The extra checkpoint is expected — a commit happened — but
	// the session count is the load-bearing number here. It stays at 2 because
	// growth extends a session rather than storing the conversation again.
	assertQueryContains(t, env, "SELECT count(*) as n FROM sessions", `"n":2`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoints", `"n":3`)
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoint_sessions", `"n":3`)
}

func TestPush_E2E_ExportAndPush(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	branch := "rekal/test@rekal.dev"

	// Checkpoint (DuckDB only).
	cleanup := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	_, _, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	// Create bare remote.
	bareDir := t.TempDir()
	bareDir, _ = filepath.EvalSymlinks(bareDir)
	initBareRemote(t, bareDir)
	if err := exec.Command("git", "-C", env.RepoDir, "remote", "add", "origin", bareDir).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	// Push should export DuckDB → wire format → orphan branch → remote.
	_, stderr, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stderr, "pushed to origin/"+branch) {
		t.Errorf("expected push success message, got: %q", stderr)
	}

	// Verify wire format on orphan branch after push.
	body1 := gitShow(env.RepoDir, branch, "rekal.body")
	dict1 := gitShow(env.RepoDir, branch, "dict.bin")
	if body1 == nil || len(body1) <= 9 {
		t.Fatal("body should have frames after push")
	}
	if dict1 == nil || len(dict1) <= 12 {
		t.Fatal("dict should have entries after push")
	}

	// Scan frames — expect 3: session + checkpoint + meta.
	frames1, err := codec.ScanFrames(body1)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	// The checkpoint's session + checkpoint frames are grouped into one batch
	// frame (per-push batching), so expect 2: batch + meta.
	if len(frames1) != 2 {
		t.Fatalf("expected 2 frames after push (batch + meta), got %d", len(frames1))
	}
	if frames1[0].Type != codec.FrameBatch {
		t.Errorf("frame 0: expected batch (0x06), got 0x%02x", frames1[0].Type)
	}
	if frames1[1].Type != codec.FrameMeta {
		t.Errorf("frame 1: expected meta (0x03), got 0x%02x", frames1[1].Type)
	}

	dec, err := codec.NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	// Decode the batch into its member frames: session + checkpoint.
	members, err := dec.DecodeBatch(codec.ExtractFramePayload(body1, frames1[0]))
	if err != nil {
		t.Fatalf("DecodeBatch: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("batch members: got %d, want 2 (session + checkpoint)", len(members))
	}
	if members[0].Type != codec.FrameSession {
		t.Errorf("member 0: expected session (0x01), got 0x%02x", members[0].Type)
	}
	if members[1].Type != codec.FrameCheckpoint {
		t.Errorf("member 1: expected checkpoint (0x02), got 0x%02x", members[1].Type)
	}

	// Decode the session member and verify data.
	sf, err := codec.DecodeSessionPayload(members[0].Payload)
	if err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if len(sf.Turns) != 5 {
		t.Errorf("session turns: got %d, want 5", len(sf.Turns))
	}
	if len(sf.ToolCalls) != 3 {
		t.Errorf("session tool_calls: got %d, want 3", len(sf.ToolCalls))
	}
	if sf.Turns[0].Text != "fix the auth bug in login.go" {
		t.Errorf("turn 0 text: %q", sf.Turns[0].Text)
	}
	if sf.ToolCalls[0].Tool != codec.ToolRead {
		t.Errorf("tool 0: got %d, want Read (%d)", sf.ToolCalls[0].Tool, codec.ToolRead)
	}

	// Decode the checkpoint member — verify session_refs.
	cf, err := codec.DecodeCheckpointPayload(members[1].Payload)
	if err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if len(cf.SessionRefs) != 1 {
		t.Errorf("checkpoint session_refs: got %d, want 1", len(cf.SessionRefs))
	}

	// Verify checkpoint is now marked exported.
	assertQueryContains(t, env, "SELECT exported FROM checkpoints", `"exported":true`)

	// Load dict and verify entries.
	loadedDict, err := codec.LoadDict(dict1)
	if err != nil {
		t.Fatalf("LoadDict: %v", err)
	}
	if loadedDict.Len(codec.NSSessions) < 1 {
		t.Errorf("dict sessions: %d", loadedDict.Len(codec.NSSessions))
	}

	// Push again — should be no-op.
	_, stderr2, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push (noop): %v", err)
	}
	if strings.Contains(stderr2, "pushed to origin/") {
		t.Errorf("second push should be no-op, got: %q", stderr2)
	}

	// --- Second checkpoint + push: append-only ---

	cleanup2 := writeSessionFile(t, env.RepoDir, "session2.jsonl", testSessionJSONL2)
	defer cleanup2()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { log.Println(\"ok\"); return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "add logging")

	_, _, err = env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint 2: %v", err)
	}

	_, _, err = env.RunCLI("push")
	if err != nil {
		t.Fatalf("push 2: %v", err)
	}

	body2 := gitShow(env.RepoDir, branch, "rekal.body")
	dict2 := gitShow(env.RepoDir, branch, "dict.bin")

	// Verify append-only: first N bytes of body2 must equal body1.
	if len(body2) <= len(body1) {
		t.Fatalf("body should grow: was %d, now %d", len(body1), len(body2))
	}
	prefix := body2[:len(body1)]
	if sha256Hex(prefix) != sha256Hex(body1) {
		t.Error("append-only violation: body prefix changed after second push")
	}

	// Each push appends one batch (its checkpoint's frames) plus a meta
	// frame, so after two pushes the body has 4 frames.
	frames2, err := codec.ScanFrames(body2)
	if err != nil {
		t.Fatalf("ScanFrames 2: %v", err)
	}
	if len(frames2) != 4 {
		t.Fatalf("expected 4 frames after second push, got %d", len(frames2))
	}

	// Dict should have grown.
	loadedDict2, err := codec.LoadDict(dict2)
	if err != nil {
		t.Fatalf("LoadDict 2: %v", err)
	}
	if loadedDict2.Len(codec.NSSessions) < 2 {
		t.Errorf("dict sessions after 2nd push: %d", loadedDict2.Len(codec.NSSessions))
	}

	t.Logf("E2E: body %d → %d bytes, dict %d → %d bytes, 4 frames, 2 sessions",
		len(body1), len(body2), len(dict1), len(dict2))
}

func TestPush_E2E_ForceOnConflict(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	cleanup := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth")

	_, _, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	bareDir := t.TempDir()
	bareDir, _ = filepath.EvalSymlinks(bareDir)
	initBareRemote(t, bareDir)
	if err := exec.Command("git", "-C", env.RepoDir, "remote", "add", "origin", bareDir).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	_, _, err = env.RunCLI("push")
	if err != nil {
		t.Fatalf("initial push: %v", err)
	}

	branch := "rekal/test@rekal.dev"

	// Simulate divergence.
	cloneDir := t.TempDir()
	cloneDir, _ = filepath.EvalSymlinks(cloneDir)
	if err := exec.Command("git", "clone", bareDir, cloneDir).Run(); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	for _, kv := range [][2]string{
		{"user.email", "other@rekal.dev"},
		{"user.name", "Other User"},
		{"gc.auto", "0"},
	} {
		exec.Command("git", "-C", cloneDir, "config", kv[0], kv[1]).Run()
	}
	exec.Command("git", "-C", cloneDir, "fetch", "origin", branch).Run()
	exec.Command("git", "-C", cloneDir, "checkout", "-b", branch, "origin/"+branch).Run()
	exec.Command("git", "-C", cloneDir, "commit", "--allow-empty", "--amend", "-m", "divergent").Run()
	exec.Command("git", "-C", cloneDir, "push", "--force", "origin", branch).Run()

	// Second checkpoint + push should detect conflict.
	cleanup2 := writeSessionFile(t, env.RepoDir, "session2.jsonl", testSessionJSONL2)
	defer cleanup2()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { log.Println(\"ok\"); return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "add logging")

	_, _, err = env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint 2: %v", err)
	}

	_, stderr, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push (conflict): %v", err)
	}
	if !strings.Contains(stderr, "non-fast-forward") && !strings.Contains(stderr, "rejected") {
		if strings.Contains(stderr, "pushed to origin/") {
			t.Errorf("conflicting push should not succeed without --force, got: %q", stderr)
		}
	}

	// Force push should succeed.
	_, stderrForce, err := env.RunCLI("push", "--force")
	if err != nil {
		t.Fatalf("push --force: %v", err)
	}
	if !strings.Contains(stderrForce, "force pushed to origin/"+branch) {
		t.Errorf("expected force push success message, got: %q", stderrForce)
	}

	localOut, _ := exec.Command("git", "-C", env.RepoDir, "rev-parse", branch).Output()
	remoteOut, _ := exec.Command("git", "-C", bareDir, "rev-parse", branch).Output()
	if strings.TrimSpace(string(localOut)) != strings.TrimSpace(string(remoteOut)) {
		t.Error("local and remote should match after force push")
	}
}

func TestPush_NoBranch_Silent(t *testing.T) {
	env := NewTestEnv(t)

	rekalDir := filepath.Join(env.RepoDir, ".rekal")
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rekalDir, "data.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push (no branch): %v", err)
	}
	if !strings.Contains(stderr, "no data to push") {
		t.Errorf("push with no branch should report no data, got: %q", stderr)
	}
}

func TestPush_NoRemote_Silent(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	_, stderr, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push (no remote): %v", err)
	}
	if !strings.Contains(stderr, "no remote") {
		t.Errorf("push with no remote should report it, got: %q", stderr)
	}
}

func TestLog_E2E(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// Log with no checkpoints — should produce no output, no error.
	stdout, _, err := env.RunCLI("log")
	if err != nil {
		t.Fatalf("log (empty): %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("log with no checkpoints should be empty, got: %q", stdout)
	}

	// Create a checkpoint.
	cleanup := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	_, _, err = env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	// Log should show the checkpoint.
	stdout, _, err = env.RunCLI("log")
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(stdout, "checkpoint ") {
		t.Errorf("log should contain 'checkpoint', got: %q", stdout)
	}
	if !strings.Contains(stdout, "Sessions: 1") {
		t.Errorf("log should show 'Sessions: 1', got: %q", stdout)
	}
	if !strings.Contains(stdout, "Author:") {
		t.Errorf("log should contain 'Author:', got: %q", stdout)
	}

	// Log --limit 0 should show nothing.
	stdout, _, err = env.RunCLI("log", "--limit", "0")
	if err != nil {
		t.Fatalf("log --limit 0: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("log --limit 0 should be empty, got: %q", stdout)
	}
}

func TestImport_E2E_RoundTrip(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// Checkpoint + push to populate orphan branch.
	cleanup := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	_, _, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	// Create bare remote and push.
	bareDir := t.TempDir()
	bareDir, _ = filepath.EvalSymlinks(bareDir)
	initBareRemote(t, bareDir)
	if err := exec.Command("git", "-C", env.RepoDir, "remote", "add", "origin", bareDir).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	// Push current branch so clone has a proper default branch.
	// Use --no-verify to skip the pre-push hook (rekal binary not in PATH during tests).
	currentBranch, _ := exec.Command("git", "-C", env.RepoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	branchName := strings.TrimSpace(string(currentBranch))
	if err := exec.Command("git", "-C", env.RepoDir, "push", "--no-verify", "origin", branchName).Run(); err != nil {
		t.Fatalf("git push %s: %v", branchName, err)
	}

	_, _, err = env.RunCLI("push")
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	// Clone into a new repo and init — should import from remote.
	cloneDir := t.TempDir()
	cloneDir, _ = filepath.EvalSymlinks(cloneDir)
	if err := exec.Command("git", "clone", bareDir, cloneDir).Run(); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	for _, kv := range [][2]string{
		{"user.email", "test@rekal.dev"},
		{"user.name", "Test User"},
		{"gc.auto", "0"},
	} {
		exec.Command("git", "-C", cloneDir, "config", kv[0], kv[1]).Run()
	}

	env2 := NewTestEnvAt(t, cloneDir)
	_, stderr, err := env2.RunCLI("init")
	if err != nil {
		t.Fatalf("init (clone): %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stderr, "imported") {
		t.Errorf("init should report imported sessions, got: %q", stderr)
	}

	// Verify DuckDB in clone has the imported data.
	assertQueryContains(t, env2, "SELECT count(*) as n FROM sessions", `"n":1`)
	assertQueryContains(t, env2, "SELECT count(*) as n FROM checkpoints", `"n":1`)

	// Log should work in the clone.
	stdout, _, err := env2.RunCLI("log")
	if err != nil {
		t.Fatalf("log (clone): %v", err)
	}
	if !strings.Contains(stdout, "checkpoint ") {
		t.Errorf("log in clone should show checkpoint, got: %q", stdout)
	}
}

func TestCheckpoint_Incremental(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// First checkpoint with session1.
	cleanup1 := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup1()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	_, stderr, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint 1: %v", err)
	}
	if !strings.Contains(stderr, "1 session(s) captured") {
		t.Errorf("expected '1 session(s) captured', got: %q", stderr)
	}

	// Second checkpoint — same file unchanged, should skip.
	gitCommit(t, env.RepoDir, "empty commit")
	_, stderr2, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint 2: %v", err)
	}
	if strings.Contains(stderr2, "session(s) captured") {
		t.Error("incremental checkpoint should skip unchanged files")
	}

	// Verify checkpoint_state table has an entry.
	assertQueryContains(t, env, "SELECT count(*) as n FROM checkpoint_state", `"n":1`)
}

// TestCheckpoint_SubagentEmbeddingDeferred verifies that incremental
// checkpoint does not synchronously nomic-embed subagent/workflow
// transcripts — only trunk sessions. Subagent fan-out (many new subagent
// transcripts discovered in one checkpoint run, e.g. after a heavy
// multi-agent session) must not make checkpoint time scale with the number
// of subagents; embedding for them is deferred to the next 'rekal index'.
// FTS indexing (BM25) still happens immediately for every session, so this
// only exercises the nomic embedding gap.
func TestCheckpoint_SubagentEmbeddingDeferred(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// Trunk session, plus a subagent transcript discovered alongside it in
	// the same checkpoint run.
	cleanupTrunk := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanupTrunk()

	sessionDir := session.FindSessionDir(env.RepoDir)
	subagentsDir := filepath.Join(sessionDir, "session1", "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("mkdir subagents dir: %v", err)
	}
	subagentJSONL := `{"parentUuid":null,"isSidechain":true,"agentId":"sub-1","type":"user","message":{"role":"user","content":"investigate the flaky test"},"uuid":"su-1","timestamp":"2026-02-25T10:00:10Z","sessionId":"test-session-001"}
{"parentUuid":"su-1","isSidechain":true,"agentId":"sub-1","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Found the race condition in the test setup."}]},"uuid":"su-2","timestamp":"2026-02-25T10:00:20Z","sessionId":"test-session-001"}
`
	if err := os.WriteFile(filepath.Join(subagentsDir, "agent-sub-1.jsonl"), []byte(subagentJSONL), 0o644); err != nil {
		t.Fatalf("write subagent file: %v", err)
	}
	defer os.RemoveAll(filepath.Join(sessionDir, "session1"))

	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")

	_, stderr, err := env.RunCLI("checkpoint")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if !strings.Contains(stderr, "2 session(s) captured") {
		t.Fatalf("expected '2 session(s) captured' (trunk + subagent), got: %q", stderr)
	}

	// Both sessions must be FTS-indexed immediately.
	stdout, _, err := env.RunCLI("query", "--index", "--json", "SELECT count(*) as n FROM session_facets")
	if err != nil {
		t.Fatalf("query session_facets: %v", err)
	}
	if !strings.Contains(stdout, `"n":2`) {
		t.Errorf("expected 2 session_facets rows, got: %s", stdout)
	}

	// Only the trunk session (null parent_session_id) should have a nomic
	// embedding after this checkpoint; the subagent's is deferred.
	trunkStdout, _, err := env.RunCLI("query", "--index", "--json",
		"SELECT count(*) as n FROM session_embeddings e JOIN session_facets f ON f.session_id = e.session_id WHERE f.parent_session_id IS NULL AND e.model = '"+nomic.ModelName+"'")
	if err != nil {
		t.Fatalf("query trunk embeddings: %v", err)
	}
	subStdout, _, err := env.RunCLI("query", "--index", "--json",
		"SELECT count(*) as n FROM session_embeddings e JOIN session_facets f ON f.session_id = e.session_id WHERE f.parent_session_id IS NOT NULL AND e.model = '"+nomic.ModelName+"'")
	if err != nil {
		t.Fatalf("query subagent embeddings: %v", err)
	}

	if !strings.Contains(subStdout, `"n":0`) {
		t.Errorf("expected 0 subagent embeddings deferred to next full index, got: %s", subStdout)
	}
	// The trunk assertion is informative rather than required: on platforms
	// without nomic support (nomic.Supported() == false) neither session
	// gets an embedding, and that is not what this test is about.
	if !strings.Contains(trunkStdout, `"n":0`) && !strings.Contains(trunkStdout, `"n":1`) {
		t.Errorf("unexpected trunk embedding count: %s", trunkStdout)
	}
}

func TestPush_NoNewCheckpoints(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	bareDir := t.TempDir()
	bareDir, _ = filepath.EvalSymlinks(bareDir)
	initBareRemote(t, bareDir)
	if err := exec.Command("git", "-C", env.RepoDir, "remote", "add", "origin", bareDir).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	// Push with no checkpoints — should report no new checkpoints but still push initial branch.
	_, stderr, err := env.RunCLI("push")
	if err != nil {
		t.Fatalf("push (no checkpoints): %v", err)
	}
	if !strings.Contains(stderr, "no new checkpoints") {
		t.Errorf("expected 'no new checkpoints' message, got: %q", stderr)
	}
	// Should still push the initial orphan branch.
	if !strings.Contains(stderr, "pushed to origin/") {
		t.Errorf("expected push of initial branch, got: %q", stderr)
	}
}

func assertQueryContains(t *testing.T, env *TestEnv, sql, expected string) {
	t.Helper()
	stdout, _, err := env.RunCLI("query", "--json", sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	if !strings.Contains(stdout, expected) {
		t.Errorf("query %q: expected %q in output, got: %q", sql, expected, stdout)
	}
}
