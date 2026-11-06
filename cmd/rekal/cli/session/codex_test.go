package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const codexFixtureJSONL = `{"type":"session_meta","session_id":"codex-001","timestamp":"2025-06-01T10:00:00Z","payload":{"cwd":"/tmp/repo","git":{"branch":"feature/auth"}}}
{"type":"event_msg","session_id":"codex-001","timestamp":"2025-06-01T10:00:01Z","payload":{"type":"user_message","message":"Add JWT authentication"}}
{"type":"event_msg","session_id":"codex-001","timestamp":"2025-06-01T10:00:05Z","payload":{"type":"agent_message","message":"I'll add JWT auth to the project."}}
{"type":"response_item","session_id":"codex-001","timestamp":"2025-06-01T10:00:06Z","payload":{"type":"function_call","name":"write_file","arguments":"{\"file_path\":\"src/auth.ts\",\"content\":\"export function verify() {}\"}"}}
{"type":"response_item","session_id":"codex-001","timestamp":"2025-06-01T10:00:07Z","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":\"npm test && echo done\"}"}}
`

func TestCodexAdapter_Parse(t *testing.T) {
	t.Parallel()

	adapter := &CodexAdapter{}

	// Write fixture to temp file.
	tmpFile := t.TempDir() + "/session.jsonl"
	if err := writeTestFile(tmpFile, codexFixtureJSONL); err != nil {
		t.Fatal(err)
	}

	payload, err := adapter.Parse(SessionRef{Path: tmpFile})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if payload.Source != "codex" {
		t.Errorf("Source = %q, want codex", payload.Source)
	}
	if payload.SessionID != "codex-001" {
		t.Errorf("SessionID = %q, want codex-001", payload.SessionID)
	}
	if payload.Branch != "feature/auth" {
		t.Errorf("Branch = %q, want feature/auth", payload.Branch)
	}

	// 2 turns: user + agent message.
	if len(payload.Turns) != 2 {
		t.Fatalf("len(Turns) = %d, want 2", len(payload.Turns))
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content != "Add JWT authentication" {
		t.Errorf("Turns[0] = %+v", payload.Turns[0])
	}
	if payload.Turns[1].Role != "assistant" || payload.Turns[1].Content != "I'll add JWT auth to the project." {
		t.Errorf("Turns[1] = %+v", payload.Turns[1])
	}

	// 2 tool calls: write_file + shell.
	if len(payload.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(payload.ToolCalls))
	}
	if payload.ToolCalls[0].Tool != "write_file" || payload.ToolCalls[0].Path != "src/auth.ts" {
		t.Errorf("ToolCalls[0] = %+v", payload.ToolCalls[0])
	}
	if payload.ToolCalls[1].Tool != "shell" || payload.ToolCalls[1].CmdPrefix != "npm test && echo done" {
		t.Errorf("ToolCalls[1] = %+v", payload.ToolCalls[1])
	}
}

func TestCodexAdapter_Parse_Empty(t *testing.T) {
	t.Parallel()

	adapter := &CodexAdapter{}
	tmpFile := t.TempDir() + "/empty.jsonl"
	if err := writeTestFile(tmpFile, ""); err != nil {
		t.Fatal(err)
	}

	payload, err := adapter.Parse(SessionRef{Path: tmpFile})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload != nil && (len(payload.Turns) != 0 || len(payload.ToolCalls) != 0) {
		t.Errorf("expected empty payload for empty file")
	}
}

func TestCodexSessionMatchesRepo(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/session.jsonl"
	if err := writeTestFile(tmpFile, codexFixtureJSONL); err != nil {
		t.Fatal(err)
	}

	if !codexSessionMatchesRepo(tmpFile, "/tmp/repo") {
		t.Error("expected session to match /tmp/repo")
	}
	if codexSessionMatchesRepo(tmpFile, "/other/repo") {
		t.Error("expected session to NOT match /other/repo")
	}
}

func TestCodexAdapter_Parse_ResponseMessages(t *testing.T) {
	t.Parallel()
	body := `{"type":"session_meta","payload":{"id":"nested-id","cwd":"/tmp/repo","git":{"branch":"fix/codex"}}}
{"type":"response_item","timestamp":"2026-11-06T10:00:00Z","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"internal instruction"}]}}
{"type":"response_item","timestamp":"2026-11-06T10:00:01Z","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Preserve the raw ledger."},{"type":"input_image","image_url":"ignored"},{"type":"input_text","text":"Test Codex capture."}]}}
{"type":"response_item","timestamp":"2026-11-06T10:00:02Z","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"go test ./cmd/rekal/cli/session\"}"}}
{"type":"response_item","timestamp":"2026-11-06T10:00:03Z","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"The adapter tests passed."}]}}
`
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := writeTestFile(path, body); err != nil {
		t.Fatal(err)
	}
	p, err := (&CodexAdapter{}).Parse(SessionRef{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if p.SessionID != "nested-id" || p.CWD != "/tmp/repo" || p.Branch != "fix/codex" {
		t.Fatalf("metadata = %+v", p)
	}
	if len(p.Turns) != 2 || p.Turns[0].Role != "human" || p.Turns[0].Content != "Preserve the raw ledger.\nTest Codex capture." || p.Turns[1].Content != "The adapter tests passed." {
		t.Fatalf("turns = %+v", p.Turns)
	}
	if p.Turns[0].Timestamp.IsZero() || p.Turns[1].Timestamp.IsZero() {
		t.Fatal("message timestamps were lost")
	}
	if len(p.ToolCalls) != 1 || p.ToolCalls[0].CmdPrefix != "go test ./cmd/rekal/cli/session" {
		t.Fatalf("tool calls = %+v", p.ToolCalls)
	}
}

func TestCodexAdapter_Parse_MirroredMessages(t *testing.T) {
	t.Parallel()
	body := `{"type":"event_msg","payload":{"type":"user_message","message":"test capture"}}
{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"test capture"}]}}
{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"captured"}]}}
{"type":"event_msg","payload":{"type":"agent_message","message":"captured"}}
{"type":"event_msg","payload":{"type":"user_message","message":"test capture"}}
`
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := writeTestFile(path, body); err != nil {
		t.Fatal(err)
	}
	p, err := (&CodexAdapter{}).Parse(SessionRef{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Turns) != 3 || p.Turns[2].Content != "test capture" {
		t.Fatalf("mirrored messages duplicated, or a genuine repeat was lost: %+v", p.Turns)
	}
}

func TestCodexAdapter_Discover_CustomHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	files := map[string]string{
		"sessions/2026/11/active.jsonl":    "/tmp/repo/subdir",
		"archived_sessions/archived.jsonl": "/tmp/repo",
		"sessions/other.jsonl":             "/tmp/repo-other",
	}
	for name, cwd := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTestFile(path, `{"type":"session_meta","payload":{"cwd":"`+cwd+`"}}`); err != nil {
			t.Fatal(err)
		}
	}
	a := &CodexAdapter{}
	refs, err := a.Discover("/tmp/repo")
	if err != nil || len(refs) != 2 {
		t.Fatalf("Discover = %v, %v; want active and archived, excluding sibling repo", refs, err)
	}
	all, err := a.DiscoverAll()
	if err != nil || len(all) != 3 {
		t.Fatalf("DiscoverAll = %v, %v; want all three custom-home sessions", all, err)
	}
}

func TestCodexAdapter_Parse_ScannerError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "too-long.jsonl")
	if err := writeTestFile(path, strings.Repeat("x", 10*1024*1024+1)); err != nil {
		t.Fatal(err)
	}
	if _, err := (&CodexAdapter{}).Parse(SessionRef{Path: path}); err == nil {
		t.Fatal("oversized record must fail rather than silently capture a partial transcript")
	}
}
