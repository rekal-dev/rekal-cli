package search

// Weights are the recall-tuning knobs: how the three retrieval layers are
// combined and how harness-metadata signals scale a hit. They are applied at
// query time only — the index stores raw signals (FTS postings, LSA and
// semantic vectors), so changing weights never requires a rebuild.
//
// Configured via the "weights" section of .rekal/config.json (see the cli
// package); the zero value is replaced by DefaultWeights.
type Weights struct {
	// Layer weights for the hybrid score. Normalized to sum to 1 before use,
	// so only their ratio matters.
	BM25     float64 // keyword precision
	LSA      float64 // corpus-specific co-occurrence
	Semantic float64 // deep semantic understanding (historical layer key "nomic")

	// SteeringBoost multiplies BM25 scores of human_steering turns — text a
	// human typed while the agent was already working is the highest-intent
	// signal in the corpus (docs/agent-metadata.md).
	SteeringBoost float64

	// SummaryBoost multiplies BM25 scores of summary turns — harness-written
	// compaction distillations. The densest recall anchors in the corpus
	// (files touched, decisions, errors and fixes, all in one turn), but
	// machine text: boosted below SteeringBoost, and kept a separate role so
	// they never masquerade as human intent.
	SummaryBoost float64

	// SubagentDownweight multiplies the hybrid score of sessions that are not
	// the trunk of their conversation (non-null parent_session_id).
	SubagentDownweight float64

	// FacetBoost scales the facet layer: BM25 over each session's facet
	// document (distinct tool paths + command prefixes + steering text; see
	// db.PopulateFacetText), added as a max-normalized fourth term —
	// hybrid += FacetBoost * facetNorm — before the subagent discount. It
	// answers structural questions ("what tools/config did session X use")
	// whose evidence lives in tool-call metadata the conversational turns
	// never mention. Ships 0.3 — the value held-out tuning selected on both
	// valid corpora (docs/research/paper); corpora with no facet material
	// pay nothing (the facet FTS index is guarded and the layer fails
	// soft). Set 0 to disable: the facet search never runs and ranking is
	// byte-identical to the pre-facet engine.
	FacetBoost float64
}

// DefaultWeights returns the tuned defaults.
func DefaultWeights() Weights {
	return Weights{
		BM25:               0.35,
		LSA:                0.10,
		Semantic:           0.55,
		SteeringBoost:      1.3,
		SummaryBoost:       1.15,
		SubagentDownweight: 0.7,
		FacetBoost:         0.3, // held-out tuned; set weights.facet_boost 0 to disable
	}
}

// orDefaults returns w, or the defaults when w is the zero value — so callers
// that never touch config get tuned behavior without thinking about it.
func (w Weights) orDefaults() Weights {
	if w == (Weights{}) {
		return DefaultWeights()
	}
	return w
}

// layers3 returns the three layer weights normalized to sum to 1, for the
// full hybrid (semantic vectors available).
func (w Weights) layers3() (bm25, lsa, nomic float64) {
	sum := w.BM25 + w.LSA + w.Semantic
	if sum <= 0 {
		d := DefaultWeights()
		return d.layers3()
	}
	return w.BM25 / sum, w.LSA / sum, w.Semantic / sum
}

// layers2 returns the BM25/LSA weights for the fallback when no semantic
// vectors are available: the semantic layer's share falls back to LSA (the
// remaining semantic signal), then the pair is normalized. With the defaults
// this yields 0.35/0.65.
func (w Weights) layers2() (bm25, lsa float64) {
	b, l, n := w.layers3()
	l += n
	sum := b + l
	return b / sum, l / sum
}
