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

func TestFuseFramings_PreservesDifferentEvidenceAnchors(t *testing.T) {
	t.Parallel()
	var outs []*search.Output
	for i, confidence := range []float64{.5, .8, .4} {
		outs = append(outs, &search.Output{Results: []search.Result{{SessionID: "session-a", Confidence: confidence, Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: i * 20, EndTurn: i * 20}}}}})
	}
	out := fuseFramings(outs)
	if len(out.Results) != 1 || out.Results[0].Confidence != .8 || len(out.Results[0].Anchors) != 3 {
		t.Fatalf("fusion lost located evidence: %+v", out)
	}
}

func TestFuseFramings_FullSeedsLeaveRoomForEachFraming(t *testing.T) {
	t.Parallel()
	var outs []*search.Output
	for i, confidence := range []float64{.5, .8, .4, .6} {
		var anchors []search.Anchor
		for j := 0; j < 4; j++ {
			anchors = append(anchors, search.Anchor{SessionID: "session-a", StartTurn: i*20 + j, EndTurn: i*20 + j})
		}
		outs = append(outs, &search.Output{Results: []search.Result{{SessionID: "session-a", Confidence: confidence, Anchors: anchors}}})
	}
	r := fuseFramings(outs).Results[0]
	seen := map[int]bool{}
	for _, a := range r.Anchors {
		seen[a.StartTurn/20] = true
	}
	if len(r.Anchors) != 4 || len(seen) != 4 || r.Anchors[0].StartTurn != 20 || r.Confidence != .8 {
		t.Fatalf("one framing consumed the entire seed: %+v", r)
	}
}
