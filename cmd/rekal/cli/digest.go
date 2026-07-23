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
		fmt.Fprintf(b, "  %s conf=%.2f t%d \"%s\"\n", sid, r.Confidence, r.SnippetTurnIdx, digestSnippet(r.Snippet))
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
		shown := len(out.Results)
		if shown > digestWindow {
			shown = digestWindow
		}
		fmt.Fprintf(&b, "INJECT top=%.2f gap=%.2f %d seeds\n", top, gap, shown)
		writeDigestRows(&b, out.Results)
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
