package search

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

type passageQueryEmbedder struct{}

func TestLexicalAnchor_LateLongTurnUsesMatchingByteWindow(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("旧的背景 ", 3000) + " uniqueevidence final correction " + strings.Repeat("后续 ", 100)
	a := lexicalAnchor(bm25Hit{sessionID: "s1", turnIndex: 42, content: text}, "uniqueevidence", 1)
	if a.StartByte == 0 || a.EndByte <= a.StartByte || !strings.Contains(text[a.StartByte:a.EndByte], "uniqueevidence") || !utf8.ValidString(text[a.StartByte:a.EndByte]) {
		t.Fatalf("late match points at the turn's opening: %+v", a)
	}
}

func TestLexicalAnchor_CaseFoldingPreservesOriginalByteOffset(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("İ", 12000) + " UNIQUEEVIDENCE final correction"
	a := lexicalAnchor(bm25Hit{sessionID: "s1", turnIndex: 42, content: text}, "uniqueevidence", 1)
	if a.EndByte <= a.StartByte || !strings.Contains(text[a.StartByte:a.EndByte], "UNIQUEEVIDENCE") || !utf8.ValidString(text[a.StartByte:a.EndByte]) {
		t.Fatalf("case-folded offset lost the original evidence: %+v", a)
	}
}

func (passageQueryEmbedder) ModelName() string                    { return "passage-test" }
func (passageQueryEmbedder) Backend() string                      { return EmbedderBackendHTTP }
func (passageQueryEmbedder) EmbedQuery(string) ([]float64, error) { return []float64{1, 0}, nil }
func (passageQueryEmbedder) Close()                               {}

func TestNomicSearch_LocatesLatePassageWithoutLexicalHit(t *testing.T) {
	t.Parallel()
	d := openTempIndexDB(t)
	seedFacet(t, d, "s1", "dev@example.com", "human")
	for i := 0; i < 20; i++ {
		text := fmt.Sprintf("unrelated context %d", i)
		if i == 19 {
			text = "the final corrected decision"
		}
		if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ($1,'s1',$2,'assistant',$3,'2026-11-06T10:00:00Z')`, fmt.Sprint(i), i, text); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.RefreshSessionPassages(d); err != nil {
		t.Fatal(err)
	}
	docs, err := db.QueryPassageDocuments(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	vectors := map[string][]float64{}
	for id, text := range docs {
		vec := []float64{0, 1}
		if strings.Contains(text, "final corrected") {
			vec = []float64{1, 0}
		}
		vectors[id] = vec
	}
	if err := db.StorePassageEmbeddings(d, vectors, "passage-test"); err != nil {
		t.Fatal(err)
	}
	// This legacy vector has no matching direction. The passage layer must
	// work even while old whole-session vectors are absent or unhelpful.
	if err := db.StoreEmbeddings(d, map[string][]float64{"s1": {0, 1}}, "passage-test"); err != nil {
		t.Fatal(err)
	}
	sem, err := nomicSearch(d, "synonym query", t.TempDir(), passageQueryEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	if sem.Scores["s1"] != 1 || len(sem.Anchors["s1"]) != 1 {
		t.Fatalf("passage not retrieved: %+v", sem)
	}
	a := sem.Anchors["s1"][0]
	if a.StartTurn != 14 || a.EndTurn != 19 {
		t.Fatalf("late evidence points elsewhere: %+v", a)
	}
	results, err := buildResults(d, []scored{{sessionID: "s1", score: 1, hit: &sessionHit{nomicScore: 1, semanticAnchors: sem.Anchors["s1"]}}}, Filters{Query: "synonym query"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].SnippetTurnIdx != 14 || results[0].MatchScope != "passage" || len(results[0].Anchors) != 1 {
		t.Fatalf("semantic pointer lost: %+v", results[0])
	}
}

func TestBuildResults_KeepsProposalRejectionAndLaterEvidence(t *testing.T) {
	t.Parallel()
	d := openTempIndexDB(t)
	seedFacet(t, d, "s1", "dev@example.com", "human")
	hit := &sessionHit{bestHit: bm25Hit{content: "initial proposal", turnIndex: 1, role: "human"}, lexicalAnchors: []Anchor{
		{SessionID: "s1", StartTurn: 1, EndTurn: 1, Kind: "lexical", Score: 3},
		{SessionID: "s1", StartTurn: 20, EndTurn: 20, Kind: "lexical", Score: 2},
		{SessionID: "s1", StartTurn: 40, EndTurn: 40, Kind: "lexical", Score: 1}}, semanticAnchors: []Anchor{{SessionID: "s1", StartTurn: 60, EndTurn: 67, Kind: "semantic", Score: .9}}}
	results, err := buildResults(d, []scored{{sessionID: "s1", score: 1, hit: hit}}, Filters{Query: "proposal"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Anchors) != 4 {
		t.Fatalf("distinct evidence collapsed: %+v", results[0].Anchors)
	}
	if got := MergeAnchors(results[0].Anchors, results[0].Anchors); len(got) != 4 {
		t.Fatal("duplicate anchors consumed the budget")
	}
}
