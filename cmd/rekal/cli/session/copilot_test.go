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
