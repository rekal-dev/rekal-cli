package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdapters_AllImplementDiscoverAll(t *testing.T) {
	t.Parallel()
	for _, a := range Adapters {
		if _, ok := a.(AllDiscoverer); !ok {
			t.Errorf("adapter %q does not implement AllDiscoverer", a.Name())
		}
	}
}

func TestContentHash_FileVsDB(t *testing.T) {
	t.Parallel()
	fileHash := ContentHash(&ClaudeAdapter{}, SessionRef{Path: "/x"}, []byte("hello"))
	if fileHash != sha256HexBytes([]byte("hello")) {
		t.Fatalf("file hash mismatch")
	}
	dbHash := ContentHash(&OpenCodeAdapter{}, SessionRef{DBID: "oc-1"}, nil)
	if dbHash != sha256HexBytes([]byte("opencode:oc-1")) {
		t.Fatalf("db hash = %q, want sha256(opencode:oc-1)", dbHash)
	}
	if fileHash == dbHash {
		t.Fatal("file and db hashes collided")
	}
}

func TestCursorAdapter_DiscoverAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoA := "/work/a"
	repoB := "/work/b"
	for _, repo := range []string{repoA, repoB} {
		stem := "sess-" + filepath.Base(repo)
		base := filepath.Join(home, ".cursor", "projects", SanitizeCursorRepoPath(repo), "agent-transcripts", stem)
		if err := os.MkdirAll(base, 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"role":"user","message":{"content":[{"type":"text","text":"hi ` + repo + `"}]}}
{"role":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}
`
		if err := writeTestFile(filepath.Join(base, stem+".jsonl"), body); err != nil {
			t.Fatal(err)
		}
	}

	refs, err := (&CursorAdapter{}).DiscoverAll()
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("DiscoverAll = %d refs, want 2", len(refs))
	}
}

func TestCopilotAdapter_DiscoverAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("COPILOT_HOME", home)
	for _, id := range []string{"a", "b"} {
		dir := filepath.Join(home, "session-state", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"` + id + `","context":{"cwd":"/work/` + id + `"}}}
{"type":"user.message","timestamp":"2026-05-07T10:00:01Z","data":{"content":"hi"}}
`
		if err := writeTestFile(filepath.Join(dir, "events.jsonl"), body); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := (&CopilotAdapter{}).DiscoverAll()
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("DiscoverAll = %d, want 2", len(refs))
	}
}
