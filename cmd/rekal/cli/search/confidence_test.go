package search

import "testing"

func TestMeaningfulQuery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		q    string
		want bool
	}{
		{"", false},
		{" ", false},
		{"\t\n", false},
		{"a", false},
		{" a ", false},
		{"ab", true},
		{"JWT expiry", true},
		{"  token  refresh ", true},
	}
	for _, tc := range cases {
		if got := MeaningfulQuery(tc.q); got != tc.want {
			t.Errorf("MeaningfulQuery(%q)=%v, want %v", tc.q, got, tc.want)
		}
	}
}

// TestRun_ThinTextQuerySilences covers BUG 3: an explicit empty/whitespace
// query arg must not fall through to filterSearch (which returns score-0 rows).
func TestRun_ThinTextQuerySilences(t *testing.T) {
	t.Parallel()
	indexDB := seedLineageCorpus(t)

	for _, tc := range []struct {
		name  string
		query string
		text  bool
	}{
		{"empty_arg", "", true},
		{"whitespace", "   ", true},
		{"single_char", "a", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Run(indexDB, Filters{Query: tc.query, TextQuery: tc.text}, t.TempDir(), DefaultWeights(), stubEmbedder{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out.Total != 0 || len(out.Results) != 0 || len(out.Knowledge) != 0 {
				t.Fatalf("thin text query must silence, got total=%d results=%+v knowledge=%+v", out.Total, out.Results, out.Knowledge)
			}
			if out.Mode != "hybrid" {
				t.Fatalf("mode=%q, want hybrid", out.Mode)
			}
		})
	}

	// Filter-only (no text arg) still returns rows.
	out, err := Run(indexDB, Filters{TextQuery: false, Limit: 5}, t.TempDir(), DefaultWeights(), stubEmbedder{})
	if err != nil {
		t.Fatalf("filter-only Run: %v", err)
	}
	if out.Total == 0 {
		t.Fatal("filter-only recall should return sessions")
	}
}

func TestAbsoluteConfidence_JunkVsReal(t *testing.T) {
	t.Parallel()
	w := DefaultWeights()
	// Evidence from scoring-lineage: real BM25 ~5.64, junk ~2.59, offtopic ~3.06.
	real := absoluteConfidence(5.64, 0, 0, 5.99, w, "")
	junk := absoluteConfidence(2.59, 0, 0, 2.40, w, "")
	off := absoluteConfidence(3.06, 0, 0, 2.90, w, "")
	if real < 0.70 {
		t.Fatalf("real confidence=%v, want >= 0.70 (got formula max(bm25Abs)+facet)", real)
	}
	if junk >= 0.70 {
		t.Fatalf("junk confidence=%v, want < 0.70", junk)
	}
	if off >= 0.70 {
		t.Fatalf("offtopic confidence=%v, want < 0.70", off)
	}
	if !(real > junk && real > off) {
		t.Fatalf("real=%v should outrank junk=%v offtopic=%v", real, junk, off)
	}
}

func TestSaturate(t *testing.T) {
	t.Parallel()
	if saturate(0, 4) != 0 {
		t.Fatal("zero")
	}
	if saturate(4, 4) < 0.6 || saturate(4, 4) > 0.7 {
		t.Fatalf("tau half-life: got %v", saturate(4, 4))
	}
}
