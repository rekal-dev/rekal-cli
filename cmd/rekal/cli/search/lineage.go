package search

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"
)

// LineageSchemaVersion is the NDJSON envelope "v" field. Bump when event
// shapes change in a breaking way.
const LineageSchemaVersion = 1

// Lineage records observe-only scoring lineage and stage timings for a
// recall query. Nil means disabled — ranking stays byte-identical to a run
// without lineage and no timing or event work runs. Wired from the
// global-only scoring_lineage config (never a CLI flag), so stdout JSON
// stays agent-clean and the diagnostic stream never leaves the machine.
//
// One recall produces events sharing RunID(): query (start) → candidate*
// (mid) → result (end). Consumers join on run_id.
type Lineage interface {
	RunID() string
	// MaxCandidates caps how many per-session candidate events are emitted
	// (top of the pre-group ranked pool).
	MaxCandidates() int
	EmitQuery(q LineageQuery)
	EmitCandidate(c LineageCandidate)
	// StageResult stores the final result body; FlushResult emits it with
	// payload_bytes filled in (stdout JSON size). Flush once per run.
	StageResult(r LineageResult)
	FlushResult(payloadBytes int)
}

// NDJSONLineage writes enveloped NDJSON events (one object per line).
// Destination is typically a lumberjack-rotated file under ~/.config/rekal/
// or stderr when no path is configured.
type NDJSONLineage struct {
	enc           *json.Encoder
	runID         string
	maxCandidates int
	staged        *LineageResult
	flushed       bool
}

// NewNDJSONLineage returns a lineage recorder writing to w. maxCandidates
// defaults to 50 when <= 0. runID is a fresh per-recall id.
func NewNDJSONLineage(w io.Writer, maxCandidates int) *NDJSONLineage {
	if maxCandidates <= 0 {
		maxCandidates = 50
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONLineage{
		enc:           enc,
		runID:         newLineageRunID(),
		maxCandidates: maxCandidates,
	}
}

func newLineageRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}

func (l *NDJSONLineage) RunID() string {
	if l == nil {
		return ""
	}
	return l.runID
}

func (l *NDJSONLineage) MaxCandidates() int {
	if l == nil {
		return 0
	}
	return l.maxCandidates
}

func (l *NDJSONLineage) EmitQuery(q LineageQuery) {
	l.emit("query", q)
}

func (l *NDJSONLineage) EmitCandidate(c LineageCandidate) {
	l.emit("candidate", c)
}

func (l *NDJSONLineage) StageResult(r LineageResult) {
	if l == nil {
		return
	}
	cp := r
	l.staged = &cp
}

func (l *NDJSONLineage) FlushResult(payloadBytes int) {
	if l == nil || l.flushed || l.staged == nil {
		return
	}
	r := *l.staged
	if r.Tokens == nil {
		r.Tokens = &LineageTokens{}
	}
	r.Tokens.PayloadBytes = payloadBytes
	l.emit("result", r)
	l.flushed = true
	l.staged = nil
}

// emit wraps body fields in the standard envelope: ts, v, run_id, event.
func (l *NDJSONLineage) emit(event string, body any) {
	if l == nil || l.enc == nil {
		return
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return
	}
	delete(fields, "event") // body must not override envelope
	out := make(map[string]any, len(fields)+4)
	out["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	out["v"] = LineageSchemaVersion
	out["run_id"] = l.runID
	out["event"] = event
	for k, v := range fields {
		out[k] = v
	}
	_ = l.enc.Encode(out) // best-effort diagnostic; never fail the recall
}

// --- Event body shapes (merged under the envelope) ---

// LineageQuery is the start-of-run snapshot: what was asked and how ranking
// is configured. Timings, returned rows, and whether the neural layer
// actually contributed (use_nomic) live on LineageResult — useNomic is only
// known after nomicSearch runs.
type LineageQuery struct {
	Query             string             `json:"query"`
	Mode              string             `json:"mode"`
	Filters           map[string]string  `json:"filters"`
	Weights           LineageWeights     `json:"weights"`
	WeightsNormalized LineageNormWeights `json:"weights_normalized"`
	EmbedderModel     string             `json:"embedder_model,omitempty"`
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

// LineageResult is the end-of-run event: final returned set, pool counts,
// stage timings, whether the neural layer contributed, and token/byte cost.
type LineageResult struct {
	Returned  []LineageReturned `json:"returned"`
	Counts    map[string]int    `json:"counts"`
	TimingsMS map[string]int64  `json:"timings_ms"`
	UseNomic  bool              `json:"use_nomic"`
	Tokens    *LineageTokens    `json:"tokens,omitempty"`
	Skipped   map[string]string `json:"skipped,omitempty"`
}

// LineageReturned is one top-level row after conversation grouping.
type LineageReturned struct {
	Rank           int     `json:"rank"`
	SessionID      string  `json:"session_id"`
	Score          float64 `json:"score"`
	SnippetTurnIdx int     `json:"snippet_turn_index"`
	SnippetRole    string  `json:"snippet_role,omitempty"`
	Children       int     `json:"children,omitempty"`
}

// LineageTokens accounts for embedding input size and stdout payload size.
type LineageTokens struct {
	EmbedQueryChars int `json:"embed_query_chars,omitempty"`
	PayloadBytes    int `json:"payload_bytes,omitempty"`
}

// lineageOn is the hot-path nil check.
func lineageOn(l Lineage) bool { return l != nil }
