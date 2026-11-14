package search

import (
	"fmt"
	"strings"
	"testing"
)

func TestLiteralPaginationAndInvalidation(t *testing.T) {
	d := openTempIndexDB(t)
	seedFacet(t, d, "session-a", "alice", "human")
	seedFacet(t, d, "session-b", "bob", "agent")
	for i := 0; i < 7; i++ {
		content := "use 100%_literal safely"
		if i == 6 {
			content = "unrelated"
		}
		if _, err := d.Exec(`INSERT INTO turns_ft VALUES (?,?,?,?,?,?)`, fmt.Sprint(i), "session-a", i, "human", content, "2026-11-14"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec(`INSERT INTO turns_ft VALUES ('b','session-b',0,'assistant','100%_literal','2026-11-14')`); err != nil {
		t.Fatal(err)
	}
	f := Filters{Query: "100%_literal", Author: "alice", Limit: 2, LimitExplicit: true}
	page, err := LiteralMatches(d, f, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 6 || len(page.Hits) != 2 || page.Next == "" {
		t.Fatalf("bad coverage: %+v", page)
	}
	initialCursor := page.Next
	seen := map[string]bool{}
	for {
		for _, h := range page.Hits {
			if seen[h.Ref] {
				t.Fatal("duplicate page result")
			}
			seen[h.Ref] = true
		}
		if page.Next == "" {
			break
		}
		page, err = LiteralMatches(d, Filters{}, page.Next)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 6 {
		t.Fatal("missing literal turns")
	}
	if _, err := d.Exec(`INSERT INTO turns_ft VALUES ('new','session-a',9,'human','100%_literal','2026-11-14')`); err != nil {
		t.Fatal(err)
	}
	if _, err := LiteralMatches(d, Filters{}, initialCursor); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("stale continuation accepted: %v", err)
	}
	zero, err := LiteralMatches(d, Filters{Query: "100%_literal", LimitExplicit: true}, "")
	if err != nil || len(zero.Hits) != 0 || zero.Next != "" {
		t.Fatal("explicit zero limit ignored")
	}
	for _, cursor := range []string{"bad!", "e30"} {
		if _, err := LiteralMatches(d, Filters{}, cursor); err == nil {
			t.Fatal("invalid cursor accepted")
		}
	}
}

func TestLiteralReferenceLocatesFullPhrase(t *testing.T) {
	d := openTempIndexDB(t)
	seedFacet(t, d, "session-a", "alice", "human")
	content := "retry " + strings.Repeat("旧背景 ", 5000) + "retry final decision"
	if _, err := d.Exec(`INSERT INTO turns_ft VALUES ('id','session-a',42,'human',?,'2026-11-14')`, content); err != nil {
		t.Fatal(err)
	}
	page, err := LiteralMatches(d, Filters{Query: "retry final decision"}, "")
	if err != nil || len(page.Hits) != 1 {
		t.Fatalf("literal match: %+v %v", page, err)
	}
	if !strings.Contains(page.Hits[0].Ref, "@") {
		t.Fatal("long-turn match lost byte location")
	}
}
