package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownScanFailsClosed(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "note.md")
	if err := os.WriteFile(note, []byte("a note"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := (MarkdownAdapter{}).Read(dir)
	if err != nil || len(files) != 1 || files[0].Content != "a note" || files[0].ModifiedAt == "" {
		t.Fatalf("scan: %+v %v", files, err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "secret.md"), filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err = (MarkdownAdapter{}).Read(dir); err == nil {
		t.Fatal("symlink scan must fail, not imply deletion")
	}
	if _, err = (MarkdownAdapter{}).Read(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing source must fail")
	}
	if err := os.Remove(filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte(strings.Repeat("a", MaxFileBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = (MarkdownAdapter{}).Read(dir); err == nil {
		t.Fatal("oversized note accepted")
	}
	if err := os.WriteFile(note, []byte{0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = (MarkdownAdapter{}).Read(dir); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func TestRevisionRejectsUnknownOrChangedPayload(t *testing.T) {
	r := Revision{Schema: 1, CheckpointID: "checkpoint", GitSHA: "abc", ObservedAt: "2026-11-14T00:00:00Z", Source: Source{Name: "notes", Adapter: "markdown", Owner: "alice", Profile: "default"}, Path: "note.md", Content: "hello"}
	r.ID = r.Digest()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Content = "changed"
	if r.Validate() == nil {
		t.Fatal("changed revision accepted")
	}
	r.ID = r.Digest()
	r.Schema = 2
	r.ID = r.Digest()
	if r.Validate() == nil {
		t.Fatal("unknown schema accepted")
	}
	r.Schema = 1
	r.Path = "../secret"
	r.ID = r.Digest()
	if r.Validate() == nil {
		t.Fatal("traversal path accepted")
	}
}

func TestClaudeMetadataDoesNotInventTimeOrType(t *testing.T) {
	for _, tc := range []struct{ input, modified, kind string }{
		{"---\nmodified: 2026-04-02T10:00:00Z\ntype: project\n---\nhello", "2026-04-02T10:00:00Z", "project"},
		{"---\nmodified: '2026-04-02T12:00:00+02:00'\ntype: \"feedback\"\n---\nhello", "2026-04-02T10:00:00Z", "feedback"},
		{"---\nmodified: today\ntype: custom\n---\nhello", "", ""},
		{"---\nmodified: 2026-04-02T10:00:00Z\nmodified: 2026-04-03T10:00:00Z\n---\nhello", "", ""},
		{"ordinary Markdown", "", ""},
	} {
		modified, kind := claudeMetadata(tc.input)
		if modified != tc.modified || kind != tc.kind {
			t.Fatalf("metadata %q => %q %q", tc.input, modified, kind)
		}
	}
}

func TestSourceScopeIdentityAndLegacyConfiguration(t *testing.T) {
	s := Source{Name: "notes", Adapter: "markdown", Owner: "alice", Profile: "work", Scope: "repo"}
	id := s.ID()
	s.Scope = "local"
	s.Shared = true
	if s.ID() != id {
		t.Fatal("publication policy changed source identity")
	}
	for _, tc := range []struct{ input, scope string }{
		{`{"name":"notes","share":false}`, "local"},
		{`{"name":"notes","share":true}`, "repo"},
		{`{"name":"notes","scope":"local","share":true}`, "local"},
	} {
		var c Config
		if err := json.Unmarshal([]byte(tc.input), &c); err != nil || c.Scope != tc.scope {
			t.Fatalf("legacy config: %+v %v", c, err)
		}
		b, err := json.Marshal(c)
		if err != nil || strings.Contains(string(b), "share") {
			t.Fatalf("legacy switch persisted: %s %v", b, err)
		}
	}
}
