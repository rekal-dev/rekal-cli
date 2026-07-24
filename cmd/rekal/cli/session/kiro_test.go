package session

import (
	"os"
	"path/filepath"
	"testing"
)

// kiroFixtureJSONL mirrors the Kiro CLI v3 event log: a Prompt (human) and an
// AssistantMessage whose content blocks carry text plus a best-effort tool
// block, and a non-conversational event kind that is skipped.
const kiroFixtureJSONL = `{"kind":"Prompt","data":{"content":[{"kind":"text","data":"Add JWT authentication"}]}}
{"kind":"AssistantMessage","data":{"content":[{"kind":"text","data":"I'll add JWT auth."},{"kind":"toolUse","data":{"name":"fs_write","input":{"path":"src/auth.ts"}}}]}}
{"kind":"ToolResult","data":{"content":[{"kind":"text","data":"ok"}]}}
`

func TestKiroAdapter_Parse(t *testing.T) {
	t.Parallel()

	tmpFile := filepath.Join(t.TempDir(), "s.jsonl")
	if err := writeTestFile(tmpFile, kiroFixtureJSONL); err != nil {
		t.Fatal(err)
	}

	payload, err := (&KiroAdapter{}).Parse(SessionRef{Path: tmpFile})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload.Source != "kiro" {
		t.Errorf("Source = %q, want kiro", payload.Source)
	}
	// Two turns: user + assistant. The tool-result line has no role → skipped.
	if len(payload.Turns) != 2 {
		t.Fatalf("Turns = %d, want 2: %+v", len(payload.Turns), payload.Turns)
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content != "Add JWT authentication" {
		t.Errorf("Turns[0] = %+v", payload.Turns[0])
	}
	if payload.Turns[1].Role != "assistant" || payload.Turns[1].Content != "I'll add JWT auth." {
		t.Errorf("Turns[1] = %+v", payload.Turns[1])
	}
	// One tool call, path extracted from the tool_use block.
	if len(payload.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d, want 1: %+v", len(payload.ToolCalls), payload.ToolCalls)
	}
	if payload.ToolCalls[0].Tool != "fs_write" || payload.ToolCalls[0].Path != "src/auth.ts" {
		t.Errorf("ToolCalls[0] = %+v, want fs_write src/auth.ts", payload.ToolCalls[0])
	}
}

func TestKiroAdapter_Discover(t *testing.T) {
	// Not parallel: uses t.Setenv.
	home := t.TempDir()
	t.Setenv("KIRO_HOME", home)

	cliDir := filepath.Join(home, "sessions", "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mk := func(id, cwd string) {
		meta := `{"session_id":"` + id + `","cwd":"` + cwd + `","title":"(no title)"}`
		if err := writeTestFile(filepath.Join(cliDir, id+".json"), meta); err != nil {
			t.Fatal(err)
		}
		if err := writeTestFile(filepath.Join(cliDir, id+".jsonl"), kiroFixtureJSONL); err != nil {
			t.Fatal(err)
		}
	}
	mk("mine", "/work/rekal/subdir") // inside the repo → match
	mk("sibling", "/work/rekal-2")   // shares prefix but is a sibling → no match
	mk("other", "/work/elsewhere")   // unrelated → no match

	refs, err := (&KiroAdapter{}).Discover("/work/rekal")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("Discover matched %d sessions, want 1: %+v", len(refs), refs)
	}
	if filepath.Base(refs[0].Path) != "mine.jsonl" {
		t.Errorf("Discover matched wrong session: %s", refs[0].Path)
	}
}

// TestKiroAdapter_DiscoverNoHome verifies a machine with no Kiro install yields
// no refs (and no error).
func TestKiroAdapter_DiscoverNoHome(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("KIRO_HOME", filepath.Join(t.TempDir(), "does-not-exist"))
	refs, err := (&KiroAdapter{}).Discover("/any/repo")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("Discover on absent home = %d refs, want 0", len(refs))
	}
}
