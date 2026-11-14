package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

func TestReferenceReadsCanonicalSourcesAndPages(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(db.StoreDir(root), 0700); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitIndexSchema(d); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO session_facets(session_id,user_email,actor_type,captured_at) VALUES ('source-b','alice','human','2026-11-14')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		if _, err := d.Exec(`INSERT INTO turns_ft VALUES (?,?,?,?,?,?)`, fmt.Sprint(i), "source-b", i, "human", fmt.Sprintf("turn %d", i), "2026-11-14"); err != nil {
			t.Fatal(err)
		}
	}
	note := memory.Revision{Schema: 2, CheckpointID: "cp", GitSHA: "sha", ObservedAt: "2026-11-14T00:00:00Z", Source: memory.Source{Name: "project", Adapter: "markdown", Owner: "alice", Profile: "default", Scope: "repo"}, Path: "note.md", Content: strings.Repeat("界", 10000)}
	note.ID = note.Digest()
	if err := db.ImportMemoryIndex(d, []memory.Revision{note}); err != nil {
		t.Fatal(err)
	}
	literal, err := search.LiteralMatches(d, search.Filters{Query: "turn", Limit: 2, LimitExplicit: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	if literal.Total != 300 || literal.Next == "" {
		t.Fatal("literal coverage missing")
	}
	digest, code := formatDigest(&search.Output{Literal: literal})
	if code != 0 || !strings.Contains(digest, "total=300") || strings.Contains(digest, "SILENCE") {
		t.Fatalf("literal-only search suppressed: %s", digest)
	}
	_ = d.Close()
	var continuation bytes.Buffer
	continueCmd := &cobra.Command{}
	continueCmd.SetOut(&continuation)
	if err := runLiteralContinuation(continueCmd, root, literal.Next, true); err != nil {
		t.Fatal(err)
	}
	var more search.Output
	if err := json.Unmarshal(continuation.Bytes(), &more); err != nil {
		t.Fatal(err)
	}
	if more.Literal.Offset != 2 || len(more.Literal.Hits) != 2 {
		t.Fatal("wrong continuation page")
	}
	read := func(ref, cursor string) referenceOutput {
		t.Helper()
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		if err := runReferenceRead(cmd, root, ref, cursor, true); err != nil {
			t.Fatal(err)
		}
		var out referenceOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	literalRead := read(more.Literal.Hits[0].Ref, "")
	if literalRead.Evidence.Windows[0].Turns[0].Index != more.Literal.Hits[0].Turn {
		t.Fatal("search reference read wrong turn")
	}
	page := read("s1", "")
	if page.Ref != "session:source-b" {
		t.Fatal("alias not canonicalized")
	}
	seen := map[int]bool{}
	for {
		for _, w := range page.Evidence.Windows {
			for _, turn := range w.Turns {
				if seen[turn.Index] {
					t.Fatal("duplicate session turn")
				}
				seen[turn.Index] = true
			}
		}
		if page.Next == "" {
			break
		}
		page = read("", page.Next)
	}
	if len(seen) != 300 {
		t.Fatalf("session lost turns: %d", len(seen))
	}
	passage := read("passage:source-b:299-299", "")
	if passage.Evidence.Windows[0].Turns[0].Content != "turn 299" {
		t.Fatal("wrong passage")
	}
	// Introducing an older sorting session moves sN, not the returned reference.
	d, err = db.OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO session_facets(session_id,user_email,actor_type,captured_at) VALUES ('source-a','alice','human','2026-11-14')`); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	if got := read(passage.Ref, ""); got.Evidence.Windows[0].SessionID != "source-b" {
		t.Fatal("canonical reference shifted")
	}
	var content strings.Builder
	page = read("memory:"+note.ID, "")
	for {
		content.WriteString(page.Content)
		if page.Next == "" {
			break
		}
		page = read("", page.Next)
	}
	if content.String() != note.Content {
		t.Fatal("UTF-8 memory pagination lost content")
	}
	var cmd cobra.Command
	if err := runReferenceRead(&cmd, root, "memory:missing", "", true); err == nil {
		t.Fatal("missing reference accepted")
	}
}
