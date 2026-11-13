package search

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

func TestMemoryEligibilityBeforeRankingAndDrill(t *testing.T) {
	d := openTempIndexDB(t)
	if err := db.EnsureMemoryIndex(d); err != nil {
		t.Fatal(err)
	}
	makeRevision := func(name, profile string, shared bool) memory.Revision {
		r := memory.Revision{Schema: 1, CheckpointID: "cp", GitSHA: "abcdef", ObservedAt: "2026-11-14T00:00:00Z", Source: memory.Source{Name: name, Adapter: "markdown", Owner: "alice", Profile: profile, Shared: shared}, Path: "note.md", Content: strings.Repeat("界", 200) + " payment retry"}
		r.ID = r.Digest()
		return r
	}
	public := makeRevision("shared", "work", true)
	private := makeRevision("private", "work", false)
	other := makeRevision("private", "personal", false)
	for _, r := range []memory.Revision{public, private, other} {
		payload, _ := json.Marshal(r)
		if _, err := d.Exec("INSERT INTO memory_documents VALUES (?,?,?,?,?)", r.ID, r.Source.ID(), r.Path, r.Content, string(payload)); err != nil {
			t.Fatal(err)
		}
	}
	hits, err := SearchMemory(d, Filters{Query: "payment retry"}, nil)
	if err != nil || len(hits) != 1 || hits[0].Ref != "memory:"+public.ID {
		t.Fatalf("private exposed: %+v %v", hits, err)
	}
	hits, err = SearchMemory(d, Filters{Query: "payment retry"}, []string{private.Source.ID()})
	if err != nil || len(hits) != 2 {
		t.Fatalf("profile scope: %+v %v", hits, err)
	}
	for _, h := range hits {
		if !utf8.ValidString(h.Snippet) || len(h.Snippet) > 384 {
			t.Fatal("snippet budget")
		}
		if strings.Contains(h.SQL, "note.md") {
			t.Fatal("drill uses a mutable path")
		}
	}
	for _, f := range []Filters{{Query: "payment", LimitExplicit: true, Limit: 0}, {Query: "payment", Author: "bob"}, {Query: "payment", Commit: "other"}, {Query: "payment", Actor: "human"}, {Query: "payment", File: "missing"}} {
		hits, err := SearchMemory(d, f, []string{private.Source.ID()})
		if err != nil || len(hits) != 0 {
			t.Fatalf("filter ignored: %+v %+v %v", f, hits, err)
		}
	}
	if _, err := SearchMemory(d, Filters{Query: "payment", File: "["}, nil); err == nil {
		t.Fatal("bad file regex accepted")
	}
}
