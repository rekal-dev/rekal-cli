package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSessionDir_DefaultsToDotClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got := FindSessionDir("/Users/frank/projects/rekal/rekal-cli")
	want := filepath.Join(home, ".claude", "projects", "-Users-frank-projects-rekal-rekal-cli")
	if got != want {
		t.Errorf("FindSessionDir() = %q, want %q", got, want)
	}
}

func TestFindSessionDir_RespectsClaudeConfigDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom/claude-home")

	got := FindSessionDir("/Users/frank/projects/rekal/rekal-cli")
	want := "/custom/claude-home/projects/-Users-frank-projects-rekal-rekal-cli"
	if got != want {
		t.Errorf("FindSessionDir() = %q, want %q", got, want)
	}
}

func TestFindSessionFiles_TopLevelOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "sess1.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sess1", "subagents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(filepath.Join(dir, "sess1", "subagents", "agent-x1.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}
	// A non-jsonl file and a tool-results sidecar dir, like real ~/.claude data, must be ignored.
	if err := os.MkdirAll(filepath.Join(dir, "sess1", "tool-results"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(filepath.Join(dir, "sess1", "tool-results", "out.txt"), "hi"); err != nil {
		t.Fatal(err)
	}

	files, err := FindSessionFiles(dir)
	if err != nil {
		t.Fatalf("FindSessionFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1: %v", len(files), files)
	}
	if files[0] != filepath.Join(dir, "sess1.jsonl") {
		t.Errorf("files[0] = %q, want sess1.jsonl", files[0])
	}
}

func TestClaudeAdapter_Discover_FindsSubagentAndWorkflowTranscripts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "sess1.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}

	subagentsDir := filepath.Join(dir, "sess1", "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(filepath.Join(subagentsDir, "agent-x1.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}

	workflowDir := filepath.Join(subagentsDir, "workflows", "release-flow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(filepath.Join(workflowDir, "step1.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}

	// A sibling directory with no matching "<name>.jsonl" (e.g. memory/) must
	// not be mistaken for a session-stem directory.
	if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(filepath.Join(dir, "memory", "note.jsonl"), "{}"); err != nil {
		t.Fatal(err)
	}

	refs, err := discoverSessionRefs(dir)
	if err != nil {
		t.Fatalf("discoverSessionRefs: %v", err)
	}

	var trunkCount, agentCount, workflowCount int
	parentPath := filepath.Join(dir, "sess1.jsonl")
	for _, ref := range refs {
		switch {
		case ref.Path == parentPath:
			trunkCount++
			if ref.ParentPath != "" {
				t.Errorf("trunk ref has ParentPath = %q, want empty", ref.ParentPath)
			}
		case ref.Path == filepath.Join(subagentsDir, "agent-x1.jsonl"):
			agentCount++
			if ref.ParentPath != parentPath {
				t.Errorf("agent ref ParentPath = %q, want %q", ref.ParentPath, parentPath)
			}
		case ref.Path == filepath.Join(workflowDir, "step1.jsonl"):
			workflowCount++
			if ref.ParentPath != parentPath {
				t.Errorf("workflow ref ParentPath = %q, want %q", ref.ParentPath, parentPath)
			}
		default:
			t.Errorf("unexpected ref: %+v", ref)
		}
	}
	if trunkCount != 1 {
		t.Errorf("trunkCount = %d, want 1", trunkCount)
	}
	if agentCount != 1 {
		t.Errorf("agentCount = %d, want 1", agentCount)
	}
	if workflowCount != 1 {
		t.Errorf("workflowCount = %d, want 1", workflowCount)
	}
}

func TestClaudeAdapter_Parse_SubagentTranscriptTaggedAsAgent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	parentPath := filepath.Join(dir, "sess1.jsonl")
	if err := writeTestFile(parentPath, "{}"); err != nil {
		t.Fatal(err)
	}

	agentFile := filepath.Join(dir, "sess1", "subagents", "agent-x1.jsonl")
	if err := os.MkdirAll(filepath.Dir(agentFile), 0o755); err != nil {
		t.Fatal(err)
	}
	// Subagent transcript entries are marked isSidechain:true; the adapter
	// must keep them (unlike the trunk transcript, which discards them).
	content := `{"uuid":"a1","sessionId":"sub-1","timestamp":"2025-01-15T10:00:00Z","type":"assistant","message":{"role":"assistant","content":"Found the bug in parse.go."},"isSidechain":true}`
	if err := writeTestFile(agentFile, content); err != nil {
		t.Fatal(err)
	}

	adapter := &ClaudeAdapter{}
	payload, err := adapter.Parse(SessionRef{Path: agentFile, ParentPath: parentPath})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload.ActorType != "agent" {
		t.Errorf("ActorType = %q, want agent", payload.ActorType)
	}
	if payload.AgentID != "x1" {
		t.Errorf("AgentID = %q, want x1", payload.AgentID)
	}
	if payload.ParentSessionPath != parentPath {
		t.Errorf("ParentSessionPath = %q, want %q", payload.ParentSessionPath, parentPath)
	}
	if len(payload.Turns) != 1 || payload.Turns[0].Content != "Found the bug in parse.go." {
		t.Fatalf("Turns = %+v, want 1 turn with sidechain content kept", payload.Turns)
	}
}

func TestClaudeAdapter_Parse_TrunkSessionStillDropsSidechain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	parentPath := filepath.Join(dir, "sess1.jsonl")
	content := `{"uuid":"a1","sessionId":"sess1","timestamp":"2025-01-15T10:00:00Z","type":"assistant","message":{"role":"assistant","content":"inline sidechain dup"},"isSidechain":true}
{"uuid":"a2","sessionId":"sess1","timestamp":"2025-01-15T10:00:01Z","type":"assistant","message":{"role":"assistant","content":"real trunk reply"}}`
	if err := writeTestFile(parentPath, content); err != nil {
		t.Fatal(err)
	}

	adapter := &ClaudeAdapter{}
	payload, err := adapter.Parse(SessionRef{Path: parentPath})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload.ActorType != "human" {
		t.Errorf("ActorType = %q, want human", payload.ActorType)
	}
	if len(payload.Turns) != 1 || payload.Turns[0].Content != "real trunk reply" {
		t.Fatalf("Turns = %+v, want 1 turn (sidechain dropped)", payload.Turns)
	}
}

func TestSubagentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path         string
		want         string
		wantWorkflow string
	}{
		{"/x/sess1/subagents/agent-x1.jsonl", "x1", ""},
		{"/x/sess1/subagents/workflows/release-flow/step1.jsonl", "step1", "release-flow"},
	}
	for _, tt := range tests {
		if got := subagentID(tt.path); got != tt.want {
			t.Errorf("subagentID(%q) = %q, want %q", tt.path, got, tt.want)
		}
		if got := workflowName(tt.path); got != tt.wantWorkflow {
			t.Errorf("workflowName(%q) = %q, want %q", tt.path, got, tt.wantWorkflow)
		}
	}
}
