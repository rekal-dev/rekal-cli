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

	// RecencyBoost scales a recency layer: candidates are nudged by how
	// recently they were captured (session_facets.captured_at), added as a
	// min-max-normalized additive term — hybrid += RecencyBoost * recencyNorm
	// — before the subagent discount. recencyNorm is 1 for the newest
	// candidate and 0 for the oldest in the result set, so it reorders within
	// a set without changing which sessions qualify. Ships 0.15 — a gentle
	// recency prior that breaks near-ties toward recent context; it is inert
	// whenever the candidate set shares a timestamp (span 0). Set 0 to disable:
	// the captured_at lookup never runs and ranking is byte-identical. Never
	// feeds absolute confidence — a newer session is not inherently more
	// relevant.
	RecencyBoost float64

	// ReachBoost scales the L1 recall-graph layer: sessions past recalls and
	// drills have reached (index session_reach.reach_count) get a
	// max-normalized additive boost — hybrid += ReachBoost * reachNorm —
	// before the subagent discount. This turns the citation-graph signal
	// ("load-bearing memory ranks higher") from a display-only hint into
	// ranking (the L1→L2 seam in docs/design/recall-graph.md). Ships 0.2 —
	// self-activating: a cold store has no reach edges, so reachNorm is 0 for
	// every session and ranking is byte-identical until the graph accumulates.
	// The layer fails soft on an index with no reach table. Set 0 to disable
	// the lookup entirely. Never feeds absolute confidence — a well-trodden
	// session is not inherently more relevant.
	ReachBoost float64
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
		FacetBoost:         0.3,  // held-out tuned; set weights.facet_boost 0 to disable
		RecencyBoost:       0.15, // gentle recency prior; inert when candidates share a timestamp; 0 to disable
		ReachBoost:         0.2,  // favors load-bearing memory; inert until the recall graph has edges; 0 to disable
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
//
// An LSA weight of exactly 0 is honored, not overridden. Everywhere else in the
// engine a weight of 0 turns its layer off — facet_boost, recency_boost,
// reach_boost all document it — and handing the semantic share to a layer the
// operator switched off inverts that: setting lsa to 0 used to leave LSA
// carrying 61% of the ranking, more than it has when enabled. The share goes to
// the only other active layer instead. Byte-identical for any store that has
// not zeroed LSA, which is every store using the defaults.
func (w Weights) layers2() (bm25, lsa float64) {
	b, l, n := w.layers3()
	switch {
	case l == 0 && b == 0:
		// Only the semantic layer was configured and it has no vectors: there
		// is nothing left to rank on. Fall back to keyword rather than scoring
		// every candidate zero.
		return 1, 0
	case l == 0:
		return 1, 0
	case b == 0:
		return 0, 1
	}
	l += n
	sum := b + l
	return b / sum, l / sum
}
