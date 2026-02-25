package session

import (
	"testing"
)

func TestSanitizeRepoPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"/Users/frank/projects/rekal/rekal-cli", "-Users-frank-projects-rekal-rekal-cli"},
		{"/home/user/repo", "-home-user-repo"},
		{"simple", "simple"},
		{"/a/b/c", "-a-b-c"},
		{"/Users/frank/My Projects/foo", "-Users-frank-My-Projects-foo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := SanitizeRepoPath(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeRepoPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

const fixtureJSONL = `{"uuid":"a1","sessionId":"sess-001","timestamp":"2025-01-15T10:00:00Z","type":"user","message":{"role":"user","content":"Add a login page"},"cwd":"/tmp/repo","gitBranch":"main"}
{"uuid":"a2","sessionId":"sess-001","timestamp":"2025-01-15T10:00:05Z","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll create a login page for you."},{"type":"tool_use","name":"Write","input":{"file_path":"src/login.tsx","content":"export default function Login() { return <div>Login</div> }"}}]},"cwd":"/tmp/repo","gitBranch":"main"}
{"uuid":"a3","sessionId":"sess-001","timestamp":"2025-01-15T10:00:10Z","type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":"File written"}]},"cwd":"/tmp/repo","gitBranch":"main"}
{"uuid":"a4","sessionId":"sess-001","timestamp":"2025-01-15T10:00:15Z","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done. The login page is at src/login.tsx."},{"type":"tool_use","name":"Bash","input":{"command":"cd /tmp/repo && npm run build && echo done"}}]},"cwd":"/tmp/repo","gitBranch":"main"}
{"uuid":"a5","sessionId":"sess-001","timestamp":"2025-01-15T10:00:20Z","type":"file-history-snapshot","message":{},"cwd":"/tmp/repo","gitBranch":"main"}
{"uuid":"a6","sessionId":"sess-001","timestamp":"2025-01-15T10:00:25Z","type":"assistant","message":{"role":"assistant","content":"Build succeeded."},"cwd":"/tmp/repo","gitBranch":"main","isSidechain":true}
`

func TestParseTranscript(t *testing.T) {
	t.Parallel()

	payload, err := ParseTranscript([]byte(fixtureJSONL))
	if err != nil {
		t.Fatalf("ParseTranscript: %v", err)
	}

	if payload.SessionID != "sess-001" {
		t.Errorf("SessionID = %q, want %q", payload.SessionID, "sess-001")
	}
	if payload.Branch != "main" {
		t.Errorf("Branch = %q, want %q", payload.Branch, "main")
	}
	if payload.ActorType != "human" {
		t.Errorf("ActorType = %q, want %q", payload.ActorType, "human")
	}

	// Turns: 1 user prompt ("Add a login page") + 2 assistant text turns.
	// The tool_result user message has no text content → filtered out.
	// The sidechain assistant message → filtered out.
	if len(payload.Turns) != 3 {
		t.Fatalf("len(Turns) = %d, want 3", len(payload.Turns))
	}

	if payload.Turns[0].Role != "human" {
		t.Errorf("Turns[0].Role = %q, want human", payload.Turns[0].Role)
	}
	if payload.Turns[0].Content != "Add a login page" {
		t.Errorf("Turns[0].Content = %q, want %q", payload.Turns[0].Content, "Add a login page")
	}

	if payload.Turns[1].Role != "assistant" {
		t.Errorf("Turns[1].Role = %q, want assistant", payload.Turns[1].Role)
	}
	if payload.Turns[1].Content != "I'll create a login page for you." {
		t.Errorf("Turns[1].Content = %q", payload.Turns[1].Content)
	}

	if payload.Turns[2].Role != "assistant" {
		t.Errorf("Turns[2].Role = %q, want assistant", payload.Turns[2].Role)
	}
	if payload.Turns[2].Content != "Done. The login page is at src/login.tsx." {
		t.Errorf("Turns[2].Content = %q", payload.Turns[2].Content)
	}

	// Tool calls: Write + Bash (from the 2 assistant messages with tool_use blocks).
	if len(payload.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(payload.ToolCalls))
	}

	if payload.ToolCalls[0].Tool != "Write" {
		t.Errorf("ToolCalls[0].Tool = %q, want Write", payload.ToolCalls[0].Tool)
	}
	if payload.ToolCalls[0].Path != "src/login.tsx" {
		t.Errorf("ToolCalls[0].Path = %q, want src/login.tsx", payload.ToolCalls[0].Path)
	}

	if payload.ToolCalls[1].Tool != "Bash" {
		t.Errorf("ToolCalls[1].Tool = %q, want Bash", payload.ToolCalls[1].Tool)
	}
	if payload.ToolCalls[1].CmdPrefix != "cd /tmp/repo && npm run build && echo done" {
		t.Errorf("ToolCalls[1].CmdPrefix = %q", payload.ToolCalls[1].CmdPrefix)
	}
}

func TestParseTranscript_Empty(t *testing.T) {
	t.Parallel()

	payload, err := ParseTranscript([]byte(""))
	if err != nil {
		t.Fatalf("ParseTranscript empty: %v", err)
	}
	if len(payload.Turns) != 0 {
		t.Errorf("expected 0 turns, got %d", len(payload.Turns))
	}
	if len(payload.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(payload.ToolCalls))
	}
}

func TestParseTranscript_MalformedLines(t *testing.T) {
	t.Parallel()

	input := `not json at all
{"uuid":"b1","sessionId":"s1","timestamp":"2025-01-15T10:00:00Z","type":"user","message":{"role":"user","content":"hello"},"gitBranch":"dev"}
also not json`

	payload, err := ParseTranscript([]byte(input))
	if err != nil {
		t.Fatalf("ParseTranscript with bad lines: %v", err)
	}
	if len(payload.Turns) != 1 {
		t.Fatalf("expected 1 turn from valid line, got %d", len(payload.Turns))
	}
	if payload.Turns[0].Content != "hello" {
		t.Errorf("Content = %q, want hello", payload.Turns[0].Content)
	}
	if payload.Branch != "dev" {
		t.Errorf("Branch = %q, want dev", payload.Branch)
	}
}

func TestParseTranscript_CmdPrefixTruncation(t *testing.T) {
	t.Parallel()

	longCmd := ""
	for i := 0; i < 150; i++ {
		longCmd += "x"
	}

	input := `{"uuid":"c1","sessionId":"s2","timestamp":"2025-01-15T10:00:00Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"` + longCmd + `"}}]},"gitBranch":"main"}`

	payload, err := ParseTranscript([]byte(input))
	if err != nil {
		t.Fatalf("ParseTranscript: %v", err)
	}
	if len(payload.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(payload.ToolCalls))
	}
	if len(payload.ToolCalls[0].CmdPrefix) != 100 {
		t.Errorf("CmdPrefix length = %d, want 100", len(payload.ToolCalls[0].CmdPrefix))
	}
}
