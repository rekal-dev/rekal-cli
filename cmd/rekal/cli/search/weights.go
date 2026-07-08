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
	BM25  float64 // keyword precision
	LSA   float64 // corpus-specific co-occurrence
	Nomic float64 // deep semantic understanding

	// SteeringBoost multiplies BM25 scores of human_steering turns — text a
	// human typed while the agent was already working is the highest-intent
	// signal in the corpus (docs/agent-metadata.md).
	SteeringBoost float64

	// SubagentDownweight multiplies the hybrid score of sessions that are not
	// the trunk of their conversation (non-null parent_session_id).
	SubagentDownweight float64
}

// DefaultWeights returns the tuned defaults.
func DefaultWeights() Weights {
	return Weights{
		BM25:               0.35,
		LSA:                0.10,
		Nomic:              0.55,
		SteeringBoost:      1.3,
		SubagentDownweight: 0.7,
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
	sum := w.BM25 + w.LSA + w.Nomic
	if sum <= 0 {
		d := DefaultWeights()
		return d.layers3()
	}
	return w.BM25 / sum, w.LSA / sum, w.Nomic / sum
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
