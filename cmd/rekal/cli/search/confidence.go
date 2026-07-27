package search

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// Absolute confidence for gating (not ranking). Max-normalization inside a
// candidate set makes junk queries score ~1.0; the hunt gate needs a signal
// that stays low when nothing in the corpus actually matches.
//
// Formula (deliberately NOT the ranking weight split — that caps BM25 at
// ~0.35 and would keep every absolute score below a useful gate):
//
//	confidence = max(saturate(bm25), cosine(nomic|lsa)) + 0.15*saturate(facet)
//
// Tuned so scoring-lineage evidence separates real domain queries
// (BM25 raw ~5–6 → conf ~0.85) from nonsense/off-topic (raw ~2.5–3 → ~0.55).

const (
	bm25Tau       = 4.0
	facetTau      = 4.0
	facetAbsBoost = 0.15
)

// MeaningfulQuery reports whether q is thick enough to run hybrid/knowledge
// search. Empty, whitespace-only, and single-character queries must not
// invent ranked results or a knowledge block (BUG 3/4).
func MeaningfulQuery(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	// At least one alphanumeric token of length >= 2.
	var run int
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			run++
			if run >= 2 {
				return true
			}
			continue
		}
		run = 0
	}
	return false
}

var multiSpace = regexp.MustCompile(`\s+`)

// NormalizeQuery trims and collapses internal whitespace for search.
func NormalizeQuery(q string) string {
	return multiSpace.ReplaceAllString(strings.TrimSpace(q), " ")
}

// saturate maps a non-negative raw score into (0,1) without depending on
// other candidates in the result set.
func saturate(raw, tau float64) float64 {
	if raw <= 0 || tau <= 0 {
		return 0
	}
	return 1 - math.Exp(-raw/tau)
}

// absoluteConfidence is the silence-gate signal. Ranking still uses
// max-normalized Score.
//
// Note there is no "nomic available" argument: an unavailable deep-semantic
// layer arrives as nomicRaw = 0 and loses the max() on its own. Passing
// availability separately would imply the gate treats "no model" differently
// from "the model found nothing", and it does not — both are absence of
// semantic evidence, and the lexical layer still speaks for itself.
func absoluteConfidence(bm25Raw, lsaRaw, nomicRaw, facetRaw float64, w Weights, parent string) float64 {
	lexical := saturate(bm25Raw, bm25Tau)
	semantic := clamp01(nomicRaw)
	if l := clamp01(lsaRaw); l > semantic {
		semantic = l
	}
	conf := lexical
	if semantic > conf {
		conf = semantic
	}
	conf = math.Min(1, conf+facetAbsBoost*saturate(facetRaw, facetTau))
	if parent != "" {
		conf *= w.orDefaults().SubagentDownweight
	}
	return round2(conf)
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
