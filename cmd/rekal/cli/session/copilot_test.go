package session

import (
	"os"
	"path/filepath"
	"testing"
)

const copilotFixtureJSONL = `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"copilot-001","copilotVersion":"1.0.54","context":{"cwd":"/tmp/repo"},"repository":{"branch":"feature/auth"}}}
{"type":"assistant.turn_start","timestamp":"2026-05-07T10:00:01Z","data":{}}
{"type":"user.message","timestamp":"2026-05-07T10:00:02Z","data":{"content":"Add JWT authentication"}}
{"type":"assistant.message","timestamp":"2026-05-07T10:00:05Z","data":{"content":{"text":"I'll add JWT auth to the project.","length":31}}}
{"type":"tool.execution_start","timestamp":"2026-05-07T10:00:06Z","data":{"toolName":"write_file","arguments":{"file_path":"src/auth.ts","content":"export function verify() {}"}}}
{"type":"tool.execution_start","timestamp":"2026-05-07T10:00:07Z","data":{"toolName":"shell","arguments":"{\"command\":\"npm test && echo done\"}"}}
{"type":"session.shutdown","timestamp":"2026-05-07T10:00:10Z","data":{}}
`

func TestCopilotAdapter_Parse(t *testing.T) {
	t.Parallel()

	adapter := &CopilotAdapter{}
	tmpFile := filepath.Join(t.TempDir(), "events.jsonl")
	if err := writeTestFile(tmpFile, copilotFixtureJSONL); err != nil {
		t.Fatal(err)
	}

	payload, err := adapter.Parse(SessionRef{Path: tmpFile})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if payload.Source != "copilot" {
		t.Errorf("Source = %q, want copilot", payload.Source)
	}
	if payload.SessionID != "copilot-001" {
		t.Errorf("SessionID = %q, want copilot-001", payload.SessionID)
	}
	if payload.Branch != "feature/auth" {
		t.Errorf("Branch = %q, want feature/auth", payload.Branch)
	}
	if payload.CWD != "/tmp/repo" {
		t.Errorf("CWD = %q, want /tmp/repo", payload.CWD)
	}

	// Two turns: user (string content) + assistant (object-with-text content).
	if len(payload.Turns) != 2 {
		t.Fatalf("Turns = %d, want 2: %+v", len(payload.Turns), payload.Turns)
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content != "Add JWT authentication" {
		t.Errorf("Turns[0] = %+v", payload.Turns[0])
	}
	if payload.Turns[1].Role != "assistant" || payload.Turns[1].Content != "I'll add JWT auth to the project." {
		t.Errorf("Turns[1] = %+v", payload.Turns[1])
	}

	// Two tool calls: object arguments and JSON-string arguments both resolve.
	if len(payload.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %d, want 2: %+v", len(payload.ToolCalls), payload.ToolCalls)
	}
	if payload.ToolCalls[0].Tool != "write_file" || payload.ToolCalls[0].Path != "src/auth.ts" {
		t.Errorf("ToolCalls[0] = %+v, want write_file src/auth.ts", payload.ToolCalls[0])
	}
	if payload.ToolCalls[1].Tool != "shell" || payload.ToolCalls[1].CmdPrefix != "npm test && echo done" {
		t.Errorf("ToolCalls[1] = %+v, want shell 'npm test && echo done'", payload.ToolCalls[1])
	}
}

func TestCopilotAdapter_Discover(t *testing.T) {
	// Not parallel: uses t.Setenv.

	// Lay out $COPILOT_HOME/session-state/<id>/events.jsonl for two sessions in
	// different repos; only the matching cwd should be discovered.
	home := t.TempDir()
	t.Setenv("COPILOT_HOME", home)

	mk := func(id, cwd string) {
		dir := filepath.Join(home, "session-state", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"` + id + `","context":{"cwd":"` + cwd + `"}}}
{"type":"user.message","timestamp":"2026-05-07T10:00:01Z","data":{"content":"hi"}}
`
		if err := writeTestFile(filepath.Join(dir, "events.jsonl"), content); err != nil {
			t.Fatal(err)
		}
	}
	mk("mine", "/work/rekal/subdir")
	mk("other", "/work/somewhere-else")

	refs, err := (&CopilotAdapter{}).Discover("/work/rekal")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("Discover matched %d sessions, want 1: %+v", len(refs), refs)
	}
	if filepath.Base(filepath.Dir(refs[0].Path)) != "mine" {
		t.Errorf("Discover matched wrong session: %s", refs[0].Path)
	}
}

// realCopilotFixtureJSONL mirrors the real ~/.copilot/session-state/<id>/
// events.jsonl shapes (github/copilot-cli#3551): session.start carries the
// branch at context.branch; the assistant.message is tool-only (empty content
// + toolRequests); the tool.execution_start repeats the same toolCallId as the
// request; system.message holds the Copilot task prompt.
const realCopilotFixtureJSONL = `{"type":"session.start","timestamp":"2026-06-01T09:00:00Z","data":{"sessionId":"copilot-real","context":{"cwd":"/tmp/repo","gitRoot":"/tmp/repo","branch":"main","headCommit":"abc123"}}}
{"type":"system.message","timestamp":"2026-06-01T09:00:01Z","data":{"role":"system","content":"You are the GitHub Copilot CLI."}}
{"type":"assistant.message","timestamp":"2026-06-01T09:00:02Z","data":{"messageId":"m1","model":"gpt","content":"","toolRequests":[{"toolCallId":"call-1","name":"shell","arguments":{"command":"go test ./..."},"type":"function","intentionSummary":"run tests"}],"turnId":"t1","outputTokens":5}}
{"type":"tool.execution_start","timestamp":"2026-06-01T09:00:03Z","data":{"toolCallId":"call-1","toolName":"shell","arguments":{"command":"go test ./..."},"model":"gpt","turnId":"t1"}}
`

// TestCopilotAdapter_ParseReal covers the real-payload fixes: the steering
// (system) turn is ingested, a tool-only assistant turn is not dropped, tool
// calls are deduped across toolRequests and tool.execution_start, the branch
// comes from context.branch, and captured_at is the session's real start.
func TestCopilotAdapter_ParseReal(t *testing.T) {
	t.Parallel()

	adapter := &CopilotAdapter{}
	tmpFile := filepath.Join(t.TempDir(), "events.jsonl")
	if err := writeTestFile(tmpFile, realCopilotFixtureJSONL); err != nil {
		t.Fatal(err)
	}

	payload, err := adapter.Parse(SessionRef{Path: tmpFile})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Branch from data.context.branch (not the empty data.repository.branch).
	if payload.Branch != "main" {
		t.Errorf("Branch = %q, want main (from context.branch)", payload.Branch)
	}

	// captured_at is the earliest event ts (session.start), not wall clock.
	wantTS := parseTimestamp("2026-06-01T09:00:00Z")
	if !payload.CapturedAt.Equal(wantTS) {
		t.Errorf("CapturedAt = %v, want %v (session.start ts)", payload.CapturedAt, wantTS)
	}

	// Turns: a human_steering (system prompt) + a tool-only assistant turn.
	var steering, assistant *Turn
	for i := range payload.Turns {
		switch payload.Turns[i].Role {
		case "human_steering":
			steering = &payload.Turns[i]
		case "assistant":
			assistant = &payload.Turns[i]
		}
	}
	if steering == nil {
		t.Fatalf("no human_steering turn; got %+v", payload.Turns)
	}
	if steering.Content != "You are the GitHub Copilot CLI." {
		t.Errorf("steering content = %q", steering.Content)
	}
	if assistant == nil {
		t.Fatalf("tool-only assistant turn was dropped; got %+v", payload.Turns)
	}
	if assistant.Content != "[tool: shell]" {
		t.Errorf("assistant content = %q, want synthesized [tool: shell]", assistant.Content)
	}

	// ToolCalls deduped by toolCallId: one shell call, not two.
	if len(payload.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d, want 1 (deduped): %+v", len(payload.ToolCalls), payload.ToolCalls)
	}
	if payload.ToolCalls[0].Tool != "shell" || payload.ToolCalls[0].CmdPrefix != "go test ./..." {
		t.Errorf("ToolCalls[0] = %+v, want shell 'go test ./...'", payload.ToolCalls[0])
	}
}

// TestCopilotMessageText covers the defensive decoder's shapes so no branch is
// untested dead code: content as a plain string, content as an object with a
// text field, content as an array of {text} parts, and the message/text field
// fallbacks the unofficial schema may use instead of content.
func TestCopilotMessageText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data string
		want string
	}{
		{"content string", `{"content":"hello"}`, "hello"},
		{"content object text", `{"content":{"text":"obj text","length":8}}`, "obj text"},
		{"content array parts", `{"content":[{"text":"a"},{"text":"b"}]}`, "a\nb"},
		{"message field fallback", `{"message":"via message"}`, "via message"},
		{"text field fallback", `{"text":"via text"}`, "via text"},
		{"empty", `{}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := copilotMessageText([]byte(tc.data)); got != tc.want {
				t.Errorf("copilotMessageText(%s) = %q, want %q", tc.data, got, tc.want)
			}
		})
	}
}

// TestCopilotAdapter_DiscoverSibling verifies a sibling repo that merely shares
// a path prefix (repo vs repo-2) is not a match — only the exact repo or a path
// inside it.
func TestCopilotAdapter_DiscoverSibling(t *testing.T) {
	// Not parallel: uses t.Setenv.
	home := t.TempDir()
	t.Setenv("COPILOT_HOME", home)

	mk := func(id, cwd string) {
		dir := filepath.Join(home, "session-state", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"` + id + `","context":{"cwd":"` + cwd + `"}}}
`
		if err := writeTestFile(filepath.Join(dir, "events.jsonl"), content); err != nil {
			t.Fatal(err)
		}
	}
	mk("exact", "/work/PKYC-Discovery")
	mk("sibling", "/work/PKYC-Discovery-2")

	refs, err := (&CopilotAdapter{}).Discover("/work/PKYC-Discovery")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("Discover matched %d sessions, want 1 (exact only): %+v", len(refs), refs)
	}
	if filepath.Base(filepath.Dir(refs[0].Path)) != "exact" {
		t.Errorf("Discover matched wrong session: %s", refs[0].Path)
	}
}

// TestCopilotAdapter_DiscoverNoHome verifies a machine with no Copilot install
// yields no refs (and no error).
func TestCopilotAdapter_DiscoverNoHome(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("COPILOT_HOME", filepath.Join(t.TempDir(), "does-not-exist"))
	refs, err := (&CopilotAdapter{}).Discover("/any/repo")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("Discover on absent home = %d refs, want 0", len(refs))
	}
}
