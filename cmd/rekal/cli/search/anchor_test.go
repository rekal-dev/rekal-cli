package search

import "testing"

func TestBuildResults_SessionScoreDoesNotInventTurnAnchor(t *testing.T) {
	t.Parallel()
	d := openTempIndexDB(t)
	seedFacet(t, d, "semantic", "dev@example.com", "human")
	if _, err := d.Exec(`INSERT INTO turns_ft (id, session_id, turn_index, role, content) VALUES
		('opening','semantic',0,'human','Hello, unrelated opening'),
		('evidence','semantic',42,'assistant','The late decision is here')`); err != nil {
		t.Fatal(err)
	}
	results, err := buildResults(d, []scored{{sessionID: "semantic", score: .9, hit: &sessionHit{nomicScore: .9}}}, Filters{Query: "synonym query"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	r := results[0]
	if r.MatchScope != "session" || r.SnippetTurnIdx != -1 || r.Snippet != "" || r.SnippetRole != "" {
		t.Fatalf("session score invented supporting evidence: %+v", r)
	}
}

func TestGroupByConversation_PreservesSnippetTranscript(t *testing.T) {
	t.Parallel()
	d := openTempIndexDB(t)
	seedFacet(t, d, "trunk", "dev@example.com", "human")
	seedFacet(t, d, "child", "dev@example.com", "agent")
	child := Result{SessionID: "child", Score: .9, Confidence: .8, Snippet: "actual evidence", SnippetTurnIdx: 7, SnippetRole: "assistant"}
	results := groupByConversation(d, []Result{child}, map[string]string{"child": "trunk"}, 10)
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	r := results[0]
	if r.SessionID != "trunk" || r.SnippetSessionID != "child" || r.SnippetTurnIdx != 7 {
		t.Fatalf("child's turn index assigned to the trunk: %+v", r)
	}
}
