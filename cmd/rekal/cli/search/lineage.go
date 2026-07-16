package search

import (
	"encoding/json"
	"io"
	"time"
)

// Lineage records observe-only scoring lineage and stage timings for a
// recall query. Nil means disabled — ranking stays byte-identical to a run
// without lineage and no timing or event work runs. Wired from the
// global-only scoring_lineage config (never a CLI flag), so stdout JSON
// stays agent-clean and the diagnostic stream never leaves the machine.
type Lineage interface {
	Emit(event any)
	// MaxCandidates caps how many per-session candidate events are emitted
	// (top of the pre-group ranked pool).
	MaxCandidates() int
}

// NDJSONLineage writes one JSON object per Emit call, newline-delimited.
// Destination is typically stderr or a file under ~/.config/rekal/.
type NDJSONLineage struct {
	enc           *json.Encoder
	maxCandidates int
}

// NewNDJSONLineage returns a lineage recorder writing to w. maxCandidates
// defaults to 50 when <= 0.
func NewNDJSONLineage(w io.Writer, maxCandidates int) *NDJSONLineage {
	if maxCandidates <= 0 {
		maxCandidates = 50
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONLineage{enc: enc, maxCandidates: maxCandidates}
}

func (l *NDJSONLineage) Emit(event any) {
	if l == nil || l.enc == nil {
		return
	}
	_ = l.enc.Encode(event) // best-effort diagnostic; never fail the recall
}

func (l *NDJSONLineage) MaxCandidates() int {
	if l == nil {
		return 0
	}
	return l.maxCandidates
}

// --- Event shapes (stable keys for log consumers) ---

// LineageQuery is the per-query summary: weights, stage timings, pool counts.
type LineageQuery struct {
	Type              string             `json:"type"` // "query"
	TS                time.Time          `json:"ts"`
	Query             string             `json:"query"`
	Mode              string             `json:"mode"`
	Filters           map[string]string  `json:"filters"`
	Weights           LineageWeights     `json:"weights"`
	WeightsNormalized LineageNormWeights `json:"weights_normalized"`
	UseNomic          bool               `json:"use_nomic"`
	EmbedderModel     string             `json:"embedder_model,omitempty"`
	TimingsMS         map[string]int64   `json:"timings_ms"`
	Counts            map[string]int     `json:"counts"`
	Skipped           map[string]string  `json:"skipped,omitempty"`
}

// LineageWeights is the configured weight snapshot (pre-normalization).
type LineageWeights struct {
	BM25               float64 `json:"bm25"`
	LSA                float64 `json:"lsa"`
	Nomic              float64 `json:"nomic"`
	SteeringBoost      float64 `json:"steering_boost"`
	SummaryBoost       float64 `json:"summary_boost"`
	SubagentDownweight float64 `json:"subagent_downweight"`
	FacetBoost         float64 `json:"facet_boost"`
}

// LineageNormWeights is the effective layer mix actually applied.
type LineageNormWeights struct {
	BM25  float64 `json:"bm25"`
	LSA   float64 `json:"lsa"`
	Nomic float64 `json:"nomic,omitempty"` // omitted in 2-way fallback
}

// LineageCandidate is one session's score lineage through the hybrid pipeline.
type LineageCandidate struct {
	Type         string             `json:"type"` // "candidate"
	SessionID    string             `json:"session_id"`
	RankPreGroup int                `json:"rank_pre_group"`
	BestTurn     *LineageBestTurn   `json:"best_turn,omitempty"`
	BM25         LineageLayer       `json:"bm25"`
	LSA          LineageLayer       `json:"lsa"`
	Nomic        LineageLayer       `json:"nomic"`
	Facet        LineageLayer       `json:"facet"`
	Contrib      map[string]float64 `json:"contrib"`
	HybridPreSub float64            `json:"hybrid_pre_subagent"`
	Subagent     *LineageSubagent   `json:"subagent,omitempty"`
	Score        float64            `json:"score"`
}

// LineageBestTurn identifies the BM25 turn that won the session slot.
type LineageBestTurn struct {
	Index int    `json:"index"`
	Role  string `json:"role"`
	Boost string `json:"boost,omitempty"` // "steering" | "summary" | ""
}

// LineageLayer holds raw and max-normalized values for one retrieval signal.
type LineageLayer struct {
	Raw  float64 `json:"raw"`
	Norm float64 `json:"norm"`
}

// LineageSubagent records the parent discount when applied.
type LineageSubagent struct {
	Parent     string  `json:"parent"`
	Multiplier float64 `json:"multiplier"`
}

// lineageOn is the hot-path nil check.
func lineageOn(l Lineage) bool { return l != nil }
