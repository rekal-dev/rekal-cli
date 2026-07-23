package cli

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

func TestFuseFramings(t *testing.T) {
	t.Parallel()

	// Two framings with an overlapping session (s2). RRF must lift s2 (it
	// appears in both) above s1 and s3, and its reported confidence must be the
	// MAX across framings (0.8), not the first-seen (0.5). Mirrors the folded
	// seek.py fusion.
	a := &search.Output{Results: []search.Result{
		{SessionID: "s1", Confidence: 0.7},
		{SessionID: "s2", Confidence: 0.5},
	}}
	b := &search.Output{Results: []search.Result{
		{SessionID: "s2", Confidence: 0.8},
		{SessionID: "s3", Confidence: 0.4},
	}}

	fused := fuseFramings([]*search.Output{a, b})

	// RRF: s1=1/61≈0.0164, s2=1/62+1/61≈0.0325, s3=1/62≈0.0161 → s2>s1>s3.
	wantOrder := []string{"s2", "s1", "s3"}
	if len(fused.Results) != 3 {
		t.Fatalf("want 3 fused results, got %d", len(fused.Results))
	}
	for i, w := range wantOrder {
		if fused.Results[i].SessionID != w {
			t.Errorf("fused[%d] = %s, want %s (order %v)", i, fused.Results[i].SessionID, w,
				[]string{fused.Results[0].SessionID, fused.Results[1].SessionID, fused.Results[2].SessionID})
		}
	}
	if fused.Results[0].Confidence != 0.8 {
		t.Errorf("s2 confidence = %v, want 0.8 (max across framings)", fused.Results[0].Confidence)
	}
	if fused.Total != 3 {
		t.Errorf("Total = %d, want 3", fused.Total)
	}

	// Single framing is an identity (order and confidences preserved).
	single := fuseFramings([]*search.Output{a})
	if len(single.Results) != 2 || single.Results[0].SessionID != "s1" || single.Results[1].SessionID != "s2" {
		t.Errorf("single-framing fusion changed order: %+v", single.Results)
	}
}
