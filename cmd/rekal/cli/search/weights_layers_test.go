package search

import (
	"math"
	"testing"
)

// TestLayers2_ZeroLSAStaysOff pins that a weight of 0 turns its layer off.
//
// layers2 is the fallback used whenever the semantic layer produced no vectors
// — a warming daemon, a store that has never run `rekal embed`, a build with no
// model. It hands the semantic layer's share to LSA as "the remaining semantic
// signal", which is reasonable when LSA is enabled and wrong when it is not:
// an operator who set lsa to 0 was left with LSA carrying 61% of the ranking,
// six times what it has when switched on. Every other layer in the engine
// documents 0 as off (facet_boost, recency_boost, reach_boost); this one
// silently overrode it.
func TestLayers2_ZeroLSAStaysOff(t *testing.T) {
	t.Parallel()

	w := DefaultWeights()
	w.LSA = 0

	bm25, lsa := w.layers2()
	if lsa != 0 {
		t.Errorf("lsa weight is %v with lsa set to 0, want 0 — a disabled layer must not receive the semantic layer's share", lsa)
	}
	if math.Abs(bm25-1) > 1e-9 {
		t.Errorf("bm25 weight is %v, want 1 — with LSA off it is the only active layer", bm25)
	}
}

// TestLayers2_DefaultsUnchanged pins that honoring a zero costs nothing for
// every store that has not set one. The fallback mix is documented as 0.35/0.65.
func TestLayers2_DefaultsUnchanged(t *testing.T) {
	t.Parallel()

	bm25, lsa := DefaultWeights().layers2()
	if math.Abs(bm25-0.35) > 1e-9 || math.Abs(lsa-0.65) > 1e-9 {
		t.Errorf("default fallback mix is %v/%v, want 0.35/0.65 — honoring a zeroed LSA must not move the default", bm25, lsa)
	}
}

// TestLayers2_ZeroBM25StaysOff is the mirror case: a keyword weight of 0 must
// not be resurrected either.
func TestLayers2_ZeroBM25StaysOff(t *testing.T) {
	t.Parallel()

	w := DefaultWeights()
	w.BM25 = 0

	bm25, lsa := w.layers2()
	if bm25 != 0 {
		t.Errorf("bm25 weight is %v with bm25 set to 0, want 0", bm25)
	}
	if math.Abs(lsa-1) > 1e-9 {
		t.Errorf("lsa weight is %v, want 1 — with BM25 off it is the only active layer", lsa)
	}
}

// TestEffectiveMix_ReportsTheFallbackNotTheIntent pins the lineage fix.
//
// The query event emits the 3-way mix because it is written before the semantic
// layer reports whether it has vectors. When it does not, scoring uses layers2
// — so the diagnostic said lsa 0.10 while the scorer used 0.65, and anyone
// reading the lineage to explain a ranking was reading a number the engine
// never applied.
func TestEffectiveMix_ReportsTheFallbackNotTheIntent(t *testing.T) {
	t.Parallel()

	w := DefaultWeights()

	full := effectiveMix(w, true)
	b3, l3, n3 := w.layers3()
	if full.BM25 != b3 || full.LSA != l3 || full.Nomic != n3 {
		t.Errorf("with semantic vectors the effective mix is %+v, want the 3-way mix %v/%v/%v", full, b3, l3, n3)
	}

	fallback := effectiveMix(w, false)
	b2, l2 := w.layers2()
	if fallback.BM25 != b2 || fallback.LSA != l2 {
		t.Errorf("without semantic vectors the effective mix is %+v, want the 2-way fallback %v/%v", fallback, b2, l2)
	}
	if fallback.Nomic != 0 {
		t.Errorf("fallback reports a semantic weight of %v, want 0 — the layer contributed nothing", fallback.Nomic)
	}
	if fallback.LSA == l3 {
		t.Error("fallback LSA weight equals the 3-way weight — the two mixes must differ, which is the whole reason the query event's number cannot be trusted here")
	}
}
