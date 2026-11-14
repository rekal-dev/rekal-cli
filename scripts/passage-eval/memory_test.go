package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

func TestExtractiveNotesPreserveOnlySourceContent(t *testing.T) {
	sessions := []session{{ID: "s1", Date: "2023-05-08T13:56:00Z", Turns: []turn{
		{Role: "user", Text: "Alice: quartzdecision was a completed event."},
		{Role: "assistant", Text: "Bob: quartzdecision was only a proposal for me."},
		{Role: "user", Text: strings.Repeat("oversized ", 100)},
	}}}
	before, _ := json.Marshal(sessions)
	ids := map[string]string{"conv_s1:0": "D1:1", "conv_s1:1": "D1:2", "conv_s1:2": "D1:3"}
	index, err := db.OpenIndexAt(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	if err := db.InitIndexSchema(index); err != nil {
		t.Fatal(err)
	}
	support, err := buildExtractiveNotes(index, "conv", sessions, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(support) != 1 {
		t.Fatalf("notes: %v", support)
	}
	hits, err := search.SearchMemory(index, search.Filters{Query: "quartzdecision"}, nil)
	if err != nil || len(hits) != 1 {
		t.Fatalf("memory lookup: %v %v", hits, err)
	}
	var payload string
	if err := index.QueryRow(hits[0].SQL).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var r memory.Revision
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{sessions[0].Date, sessions[0].Turns[0].Text, sessions[0].Turns[1].Text, "passage:conv_s1:0-0"} {
		if !strings.Contains(r.Content, text) {
			t.Fatalf("lost source attribution: %s", text)
		}
	}
	if strings.Contains(r.Content, "oversized") || len(support[r.ID]) != 2 {
		t.Fatal("excluded text marked delivered")
	}
	again, err := buildExtractiveNotes(index, "conv", sessions, ids)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := again[r.ID]; !ok {
		t.Fatal("source-only note generation is not reproducible")
	}
	after, _ := json.Marshal(sessions)
	if string(before) != string(after) {
		t.Fatal("note generation changed raw turns")
	}
}
