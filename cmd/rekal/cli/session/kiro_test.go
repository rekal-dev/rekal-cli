package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	t.Setenv("KIRO_IDE_STORAGE", filepath.Join(t.TempDir(), "no-ide"))
	refs, err := (&KiroAdapter{}).Discover("/any/repo")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("Discover on absent home = %d refs, want 0", len(refs))
	}
}

// TestKiroAdapter_ParseIDE covers the IDE session format: a history of
// user/assistant messages (string and {type,text} content) plus an optional
// toolUses array, with captured_at from sessions.json dateCreated (unix ms).
func TestKiroAdapter_ParseIDE(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Real Kiro IDE writes dateCreated as unix milliseconds, not RFC3339.
	idx := `[{"sessionId":"s1","title":"t","dateCreated":"1753744492244","workspaceDirectory":"/work/rekal"}]`
	if err := writeTestFile(filepath.Join(dir, "sessions.json"), idx); err != nil {
		t.Fatal(err)
	}
	sess := `{"history":[
	  {"message":{"role":"user","content":[{"type":"text","text":"Add JWT auth"}]}},
	  {"message":{"role":"assistant","content":"I'll add it."},"toolUses":[{"name":"fsWrite","input":{"path":"src/auth.ts"}}]}
	]}`
	sf := filepath.Join(dir, "s1.json")
	if err := writeTestFile(sf, sess); err != nil {
		t.Fatal(err)
	}

	payload, err := (&KiroAdapter{}).Parse(SessionRef{Path: sf})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload.Source != "kiro" {
		t.Errorf("Source = %q, want kiro", payload.Source)
	}
	if len(payload.Turns) != 2 {
		t.Fatalf("Turns = %d, want 2: %+v", len(payload.Turns), payload.Turns)
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content != "Add JWT auth" {
		t.Errorf("Turns[0] = %+v", payload.Turns[0])
	}
	if payload.Turns[1].Role != "assistant" || payload.Turns[1].Content != "I'll add it." {
		t.Errorf("Turns[1] = %+v", payload.Turns[1])
	}
	if len(payload.ToolCalls) != 1 || payload.ToolCalls[0].Tool != "fsWrite" || payload.ToolCalls[0].Path != "src/auth.ts" {
		t.Fatalf("ToolCalls = %+v, want fsWrite src/auth.ts", payload.ToolCalls)
	}
	want := time.UnixMilli(1753744492244).UTC()
	if !payload.CapturedAt.Equal(want) {
		t.Errorf("CapturedAt = %v, want unix-ms dateCreated %v", payload.CapturedAt, want)
	}
}

// TestKiroAdapter_ParseIDENoToolUses matches observed local Kiro IDE sessions:
// history has message/executionId/promptLogs but no toolUses — text still parses.
func TestKiroAdapter_ParseIDENoToolUses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	idx := `[{"sessionId":"s2","dateCreated":"1753744492244","workspaceDirectory":"/work/rekal"}]`
	if err := writeTestFile(filepath.Join(dir, "sessions.json"), idx); err != nil {
		t.Fatal(err)
	}
	sess := `{"history":[
	  {"message":{"role":"user","content":[{"type":"text","text":"build a game"}]},"contextItems":[],"editorState":{}},
	  {"message":{"role":"assistant","content":"On it."},"executionId":"de6c644e","promptLogs":[]}
	]}`
	sf := filepath.Join(dir, "s2.json")
	if err := writeTestFile(sf, sess); err != nil {
		t.Fatal(err)
	}
	payload, err := (&KiroAdapter{}).Parse(SessionRef{Path: sf})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(payload.Turns) != 2 {
		t.Fatalf("Turns = %d, want 2", len(payload.Turns))
	}
	if len(payload.ToolCalls) != 0 {
		t.Errorf("ToolCalls = %+v, want none when toolUses absent", payload.ToolCalls)
	}
}

// TestKiroAdapter_LocalIDESession evidences discovery+parse against a real
// Kiro IDE install on this machine (skipped when absent).
func TestKiroAdapter_LocalIDESession(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	var root string
	switch {
	case fileExists(filepath.Join(home, "Library", "Application Support", "Kiro", "User", "globalStorage", "kiro.kiroagent")):
		root = filepath.Join(home, "Library", "Application Support", "Kiro", "User", "globalStorage", "kiro.kiroagent")
	case fileExists(filepath.Join(home, ".config", "Kiro", "User", "globalStorage", "kiro.kiroagent")):
		root = filepath.Join(home, ".config", "Kiro", "User", "globalStorage", "kiro.kiroagent")
	default:
		t.Skip("no local Kiro IDE storage")
	}
	t.Setenv("KIRO_IDE_STORAGE", root)
	t.Setenv("KIRO_HOME", filepath.Join(t.TempDir(), "no-cli"))

	ws, ok := firstLocalKiroWorkspace(root)
	if !ok {
		t.Skip("no indexed Kiro IDE workspace sessions")
	}
	refs, err := (&KiroAdapter{}).Discover(ws)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) == 0 {
		t.Fatalf("Discover(%s) matched 0 sessions despite index entry", ws)
	}
	payload, err := (&KiroAdapter{}).Parse(refs[0])
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload == nil || payload.Source != "kiro" {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Turns) == 0 {
		t.Fatal("expected conversation turns from local session")
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content == "" {
		t.Errorf("first turn = %+v", payload.Turns[0])
	}
	// Real dateCreated is unix ms — must not fall back to wall clock.
	if time.Since(payload.CapturedAt) < time.Hour {
		t.Errorf("CapturedAt %v looks like wall-clock fallback; want index dateCreated", payload.CapturedAt)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstLocalKiroWorkspace(root string) (string, bool) {
	wsRoot := filepath.Join(root, "workspace-sessions")
	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(wsRoot, e.Name(), "sessions.json"))
		if err != nil {
			continue
		}
		var index []kiroIDEIndex
		if err := json.Unmarshal(data, &index); err != nil {
			continue
		}
		for _, item := range index {
			if item.Hidden || item.WorkspaceDirectory == "" {
				continue
			}
			return item.WorkspaceDirectory, true
		}
	}
	return "", false
}

// TestKiroAdapter_DiscoverIDE verifies IDE discovery matches on the index's
// workspaceDirectory (repo or inside it), skips hidden sessions, and rejects a
// prefix-sharing sibling repo.
func TestKiroAdapter_DiscoverIDE(t *testing.T) {
	// Not parallel: uses t.Setenv.
	base := t.TempDir()
	t.Setenv("KIRO_IDE_STORAGE", base)
	t.Setenv("KIRO_HOME", filepath.Join(t.TempDir(), "no-cli")) // isolate CLI discovery

	wsDir := filepath.Join(base, "workspace-sessions", "ws1")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	idx := `[
	  {"sessionId":"mine","workspaceDirectory":"/work/rekal/sub"},
	  {"sessionId":"sibling","workspaceDirectory":"/work/rekal-2"},
	  {"sessionId":"hidden","workspaceDirectory":"/work/rekal","hidden":true}
	]`
	if err := writeTestFile(filepath.Join(wsDir, "sessions.json"), idx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"mine", "sibling", "hidden"} {
		if err := writeTestFile(filepath.Join(wsDir, id+".json"), `{"history":[]}`); err != nil {
			t.Fatal(err)
		}
	}

	refs, err := (&KiroAdapter{}).Discover("/work/rekal")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("Discover matched %d IDE sessions, want 1: %+v", len(refs), refs)
	}
	if filepath.Base(refs[0].Path) != "mine.json" {
		t.Errorf("Discover matched wrong session: %s", refs[0].Path)
	}
}
