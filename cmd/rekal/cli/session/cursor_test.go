package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeCursorRepoPath(t *testing.T) {
	t.Parallel()
	got := SanitizeCursorRepoPath("/Users/frank/projects/rekal/rekal-cli")
	want := "Users-frank-projects-rekal-rekal-cli"
	if got != want {
		t.Fatalf("SanitizeCursorRepoPath = %q, want %q", got, want)
	}
}

func TestCursorAdapter_DiscoverAndParse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := "/Users/frank/projects/rekal/rekal-cli"
	stem := "2d3113dc-40b7-4ea7-bd51-12f07e957585"
	subID := "a4a7b519-6be3-46ba-bc76-44be6cbafbb3"
	base := filepath.Join(home, ".cursor", "projects", SanitizeCursorRepoPath(repo), "agent-transcripts", stem)
	if err := os.MkdirAll(filepath.Join(base, "subagents"), 0o755); err != nil {
		t.Fatal(err)
	}

	trunk := `{"role":"user","message":{"content":[{"type":"text","text":"<timestamp>Friday, Jul 17, 2026, 6:10 PM (UTC+10)</timestamp>\n<user_query>\nhello\n</user_query>"}]}}
{"role":"assistant","message":{"content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Shell","input":{"command":"git status","description":"status"}}]}}
{"type":"turn_ended","status":"ok"}
{"role":"assistant","message":{"content":[{"type":"tool_use","name":"StrReplace","input":{"path":"cmd/rekal/cli/session/cursor.go","old_string":"a","new_string":"b"}}]}}
`
	sub := `{"role":"user","message":{"content":[{"type":"text","text":"sub task"}]}}
{"role":"assistant","message":{"content":[{"type":"text","text":"working"}]}}
`
	trunkPath := filepath.Join(base, stem+".jsonl")
	subPath := filepath.Join(base, "subagents", subID+".jsonl")
	if err := writeTestFile(trunkPath, trunk); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(subPath, sub); err != nil {
		t.Fatal(err)
	}

	adapter := &CursorAdapter{}
	refs, err := adapter.Discover(repo)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("Discover refs = %d, want 2", len(refs))
	}
	if refs[0].Path != trunkPath || refs[0].ParentPath != "" {
		t.Fatalf("trunk ref = %+v", refs[0])
	}
	if refs[1].Path != subPath || refs[1].ParentPath != trunkPath {
		t.Fatalf("sub ref = %+v", refs[1])
	}

	payload, err := adapter.Parse(refs[0])
	if err != nil {
		t.Fatalf("Parse trunk: %v", err)
	}
	if payload.Source != "cursor" {
		t.Fatalf("Source = %q, want cursor", payload.Source)
	}
	if payload.SessionID != stem {
		t.Fatalf("SessionID = %q, want %q", payload.SessionID, stem)
	}
	if payload.ActorType != "human" {
		t.Fatalf("ActorType = %q, want human", payload.ActorType)
	}
	if len(payload.Turns) != 2 {
		t.Fatalf("Turns = %d, want 2 (user + first assistant text)", len(payload.Turns))
	}
	if payload.Turns[0].Role != "human" || payload.Turns[1].Role != "assistant" {
		t.Fatalf("roles = %q/%q", payload.Turns[0].Role, payload.Turns[1].Role)
	}
	if payload.Turns[0].Timestamp.IsZero() {
		t.Fatal("expected timestamp parsed from <timestamp> wrapper")
	}
	if len(payload.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %d, want 2", len(payload.ToolCalls))
	}
	if payload.ToolCalls[0].Tool != "Shell" || payload.ToolCalls[0].CmdPrefix != "git status" {
		t.Fatalf("tool0 = %+v", payload.ToolCalls[0])
	}
	if payload.ToolCalls[1].Tool != "StrReplace" || payload.ToolCalls[1].Path != "cmd/rekal/cli/session/cursor.go" {
		t.Fatalf("tool1 = %+v", payload.ToolCalls[1])
	}

	subPayload, err := adapter.Parse(refs[1])
	if err != nil {
		t.Fatalf("Parse sub: %v", err)
	}
	if subPayload.ActorType != "agent" || subPayload.AgentID != subID {
		t.Fatalf("sub meta ActorType=%q AgentID=%q", subPayload.ActorType, subPayload.AgentID)
	}
	if subPayload.ParentSessionPath != trunkPath {
		t.Fatalf("ParentSessionPath = %q", subPayload.ParentSessionPath)
	}
}

func TestCursorAdapter_Discover_MissingDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	adapter := &CursorAdapter{}
	refs, err := adapter.Discover("/tmp/no-such-repo")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs = %d, want 0", len(refs))
	}
}
