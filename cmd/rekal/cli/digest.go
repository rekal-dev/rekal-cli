package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

// digest is the in-binary port of the skill's route.py: it turns a recall
// result into the agent-facing seed digest (INJECT / KNOWLEDGE / SILENCE +
// per-seed `conf=`), byte-identical to route.py. Labels are recommendations,
// not decisions — a super-low, env-overridable confidence floor only
// machine-silences the empty/near-zero case; the agent weighs `conf=`.

const (
	digestWindow       = 20
	digestSnippetWords = 15
)

func huntFloat(name string, def float64) float64 {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return v
}

// Floors read the same REKAL_HUNT_* overrides as route.py (research escape
// hatch; shipped default is one honest bar).
func digestConfMin() float64      { return huntFloat("REKAL_HUNT_CONF_MIN", 0.25) }
func digestConfSoft() float64     { return huntFloat("REKAL_HUNT_CONF_SOFT", 0.20) }
func digestGapMin() float64       { return huntFloat("REKAL_HUNT_GAP_MIN", 0.02) }
func digestKnowledgeMin() float64 { return huntFloat("REKAL_HUNT_KNOWLEDGE_MIN", 0.25) }

// episodeVerdict gates on absolute confidence (never max-normalized score).
// Returns kind ∈ {pass, silence, empty}, plus top/gap and a silence reason.
func episodeVerdict(results []search.Result) (kind string, top, gap float64, reason string) {
	if len(results) == 0 {
		return "empty", 0, 0, "no_results"
	}
	confs := make([]float64, len(results))
	for i, r := range results {
		confs[i] = r.Confidence
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(confs)))
	top = confs[0]
	gap = top
	if len(confs) > 1 {
		gap = confs[0] - confs[1]
	}
	switch {
	case top >= digestConfMin():
		return "pass", top, gap, ""
	case top >= digestConfSoft() && gap >= digestGapMin():
		return "pass", top, gap, ""
	case len(confs) == 1:
		return "silence", top, gap, "single_below_conf"
	default:
		return "silence", top, gap, "below_gate"
	}
}

func digestSnippet(s string) string {
	words := strings.Fields(s)
	if len(words) > digestSnippetWords {
		return strings.Join(words[:digestSnippetWords], " ") + "…"
	}
	return strings.Join(words, " ")
}

// knowledgeHits renders the top-5 knowledge files above the report floor as
// `path=score`, score-descending; empty when none qualify.
func knowledgeHits(k []search.KnowledgeHit) string {
	type kv struct {
		path  string
		score float64
	}
	var rep []kv
	floor := digestKnowledgeMin()
	for _, h := range k {
		if h.Path == "" {
			continue
		}
		if h.Score >= floor {
			rep = append(rep, kv{h.Path, h.Score})
		}
	}
	sort.SliceStable(rep, func(i, j int) bool { return rep[i].score > rep[j].score })
	var parts []string
	for i, e := range rep {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s=%.2f", e.path, e.score))
	}
	return strings.Join(parts, " ")
}

// reachQueryCap bounds the representative query echoed in the reach hint so one
// long question can't blow up the terse digest line.
const reachQueryCap = 30

// reachHint renders the L1 recall-graph suffix for a seed: how often it was
// surfaced, how often an agent actually opened it, and the query that surfaced
// it most often. Empty when the seed has no reach history, so a cold store's
// digest is byte-identical to before the feature.
//
// The drill count is printed separately, and only when there is one, because
// the two numbers are different evidence: being surfaced is this engine's own
// past output, while being opened is an agent's judgment. A high reach with no
// drills means "the ranker keeps offering this", which is not the same
// recommendation as "people read this".
func reachHint(r *search.ReachInfo) string {
	if r == nil || r.Count <= 0 {
		return ""
	}
	counts := fmt.Sprintf("reached %d×", r.Count)
	if r.Drills > 0 {
		counts += fmt.Sprintf(" drilled %d×", r.Drills)
	}
	q := strings.TrimSpace(r.Query)
	if q == "" {
		return fmt.Sprintf(" [%s]", counts)
	}
	if rs := []rune(q); len(rs) > reachQueryCap {
		q = string(rs[:reachQueryCap]) + "…"
	}
	return fmt.Sprintf(" [%s· %q]", counts, q)
}

// withEvidence drops seeds the engine scored as carrying no absolute evidence
// at all — zero confidence and zero BM25 mass together. Such a seed reaches the
// digest only on max-normalized score, which is relative by construction: when
// the whole candidate set is weak, something still normalizes to the top. On a
// real store these are harness echoes ("Reply with exactly: OK") occupying
// slots in a 20-seed window that the agent pays for and cannot use.
//
// This is not a confidence floor. It gates on exact zero — a property of the
// engine's own absolute scoring, corpus-invariant by construction — never on a
// cutoff read off a corpus. The soul forbids the second, not the first. Seeds
// with any evidence at all, however small, still render and the agent judges
// them from `conf=`. Digest only: --json stays raw.
func withEvidence(results []search.Result) []search.Result {
	kept := make([]search.Result, 0, len(results))
	for _, r := range results {
		if r.Confidence == 0 && r.Mass == 0 {
			continue
		}
		kept = append(kept, r)
	}
	return kept
}

func writeDigestRows(b *strings.Builder, results []search.Result) {
	n := len(results)
	if n > digestWindow {
		n = digestWindow
	}
	for _, r := range results[:n] {
		sid := r.Sid
		if sid == "" {
			sid = r.SessionID
		}
		// Literal double quotes around the (unescaped) snippet, as route.py.
		// The reach hint is empty unless the seed has recall-graph history, so
		// this stays byte-identical to route.py on a cold store.
		fmt.Fprintf(b, "  %s conf=%.2f t%d%s \"%s\"\n",
			sid, r.Confidence, r.SnippetTurnIdx, reachHint(r.Reached), digestSnippet(r.Snippet))
	}
	if more := len(results) - digestWindow; more > 0 {
		fmt.Fprintf(b, "  (+%d more)\n", more)
	}
}

// formatDigest returns the digest text and the route.py exit code (0 for
// INJECT/KNOWLEDGE, 1 for SILENCE). Substrates are inclusive: an INJECT may
// trail a KNOWLEDGE line.
func formatDigest(out *search.Output) (string, int) {
	var b strings.Builder
	kind, top, gap, reason := episodeVerdict(out.Results)
	warming := out.Semantic != nil && out.Semantic.Retryable

	finish := func(code int) (string, int) {
		if warming {
			fmt.Fprintln(&b, "SEMANTIC warming — keyword+LSA only; retry with backoff")
		}
		return b.String(), code
	}

	if kind == "pass" {
		// The verdict above is computed on the full result set — the gate's
		// job is unchanged. Only what the digest spends its window on is
		// filtered.
		seeds := withEvidence(out.Results)
		shown := len(seeds)
		if shown > digestWindow {
			shown = digestWindow
		}
		fmt.Fprintf(&b, "INJECT top=%.2f gap=%.2f %d seeds\n", top, gap, shown)
		writeDigestRows(&b, seeds)
		if h := knowledgeHits(out.Knowledge); h != "" {
			fmt.Fprintf(&b, "KNOWLEDGE %s\n", h)
		}
		return finish(0)
	}

	if h := knowledgeHits(out.Knowledge); h != "" {
		fmt.Fprintf(&b, "KNOWLEDGE %s\n", h)
		return finish(0)
	}

	if reason == "" {
		reason = "below_gate"
	}
	fmt.Fprintf(&b, "SILENCE reason=%s\n", reason)
	return finish(1)
}
