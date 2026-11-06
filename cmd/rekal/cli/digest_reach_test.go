package cli

import (
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

func TestReachHint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   *search.ReachInfo
		want string
	}{
		{"nil", nil, ""},
		{"zero count", &search.ReachInfo{Count: 0, Query: "x"}, ""},
		{"count only", &search.ReachInfo{Count: 3}, " [reached 3×]"},
		{"count and query", &search.ReachInfo{Count: 4, Query: "jwt expiry"}, ` [reached 4×· "jwt expiry"]`},
		{"blank query is count only", &search.ReachInfo{Count: 2, Query: "   "}, " [reached 2×]"},
		{
			"long query truncated",
			&search.ReachInfo{Count: 1, Query: "a very long question that certainly exceeds the cap"},
			` [reached 1×· "a very long question that cert…"]`,
		},
		// Drills are the signal an agent acted on; they are shown separately so
		// "the ranker keeps offering this" cannot read as "people read this".
		{"drills shown when present", &search.ReachInfo{Count: 9, Drills: 2}, " [reached 9× drilled 2×]"},
		{
			"drills with query",
			&search.ReachInfo{Count: 9, Drills: 2, Query: "jwt expiry"},
			` [reached 9× drilled 2×· "jwt expiry"]`,
		},
		{"no drills stays as before", &search.ReachInfo{Count: 9, Drills: 0}, " [reached 9×]"},
	}
	for _, c := range cases {
		if got := reachHint(c.in); got != c.want {
			t.Errorf("%s: reachHint = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestWriteDigestRows_ReachSuffix proves the suffix appears only for seeds with
// reach history, and that a seed without it renders exactly as before (no
// trailing space, byte-identical to the route.py line shape).
func TestWriteDigestRows_ReachSuffix(t *testing.T) {
	t.Parallel()
	results := []search.Result{
		{Sid: "s1", Confidence: 0.50, SnippetTurnIdx: 12, Snippet: "reached seed",
			Reached: &search.ReachInfo{Count: 4, Query: "jwt expiry"}},
		{Sid: "s2", Confidence: 0.40, SnippetTurnIdx: 7, Snippet: "cold seed"},
	}
	var b strings.Builder
	writeDigestRows(&b, results)
	got := b.String()

	wantS1 := `  s1 conf=0.50 t12 [reached 4×· "jwt expiry"] "reached seed"` + "\n"
	wantS2 := `  s2 conf=0.40 t7 "cold seed"` + "\n"
	if !strings.Contains(got, wantS1) {
		t.Errorf("missing reach-annotated line.\ngot:\n%s\nwant line:\n%s", got, wantS1)
	}
	if !strings.Contains(got, wantS2) {
		t.Errorf("cold seed line changed.\ngot:\n%s\nwant line:\n%s", got, wantS2)
	}
}

func TestWriteDigestRows_DistinguishesSessionAndChildAnchors(t *testing.T) {
	t.Parallel()
	results := []search.Result{
		{SessionID: "semantic", Sid: "s1", Confidence: .8, MatchScope: "session", SnippetTurnIdx: -1},
		{SessionID: "trunk", Sid: "s2", Confidence: .7, Snippet: "child evidence", SnippetTurnIdx: 7, SnippetSessionID: "child",
			Children: []search.Result{{SessionID: "child", Sid: "s3"}}},
	}
	var b strings.Builder
	writeDigestRows(&b, results)
	got := b.String()
	if !strings.Contains(got, `s1 conf=0.80 session ""`) || strings.Contains(got, "t-1") {
		t.Fatalf("unlocated semantic match looks like a turn hit: %s", got)
	}
	if !strings.Contains(got, `s2 conf=0.70 s3:t7 "child evidence"`) {
		t.Fatalf("digest points into the wrong transcript: %s", got)
	}
}
