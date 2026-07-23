package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/embedhttp"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/graph"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

// reachLogCap bounds how many surfaced sessions one recall records as edges —
// the returned set is already limited by --limit, but a very deep limit
// shouldn't spool an unbounded batch. Reaching past the digest window is still
// a real "surfaced" signal, so the cap is generous.
const reachLogCap = 50

// attachReach fills each result's Reached hint from the derived session_reach
// aggregate (index.db). Best-effort and display-only: on any error the results
// are left untouched and recall proceeds with no hint.
func attachReach(indexDB *sql.DB, results []search.Result) {
	if len(results) == 0 {
		return
	}
	ids := make([]string, 0, len(results))
	for _, r := range results {
		if r.SessionID != "" {
			ids = append(ids, r.SessionID)
		}
	}
	reach, err := db.LoadReach(indexDB, ids)
	if err != nil || len(reach) == 0 {
		return
	}
	for i := range results {
		if rc, ok := reach[results[i].SessionID]; ok && rc.Count > 0 {
			results[i].Reached = &search.ReachInfo{Count: rc.Count, Query: rc.Query}
		}
	}
}

// logRecallEdges spools one recall edge per surfaced session for the checkpoint
// drain. Query is the original question (framings are not logged — they are
// synthetic phrasings and would pollute the representative query). Best-effort.
func logRecallEdges(gitRoot, query string, results []search.Result) {
	if len(results) == 0 {
		return
	}
	now := time.Now().UTC()
	edges := make([]graph.Edge, 0, len(results))
	for _, r := range results {
		if r.SessionID == "" {
			continue
		}
		edges = append(edges, graph.Edge{TS: now, Kind: "recall", Query: query, Target: r.SessionID})
		if len(edges) >= reachLogCap {
			break
		}
	}
	_ = graph.Append(gitRoot, edges) //nolint:errcheck // best-effort telemetry
}

// rrfK is the standard Reciprocal Rank Fusion constant (Cormack, Clarke &
// Büttcher 2009) — a rank-smoothing convention, not a corpus-tuned number, and
// it decides nothing (fusion only reorders). Same value as the folded seek.py.
const rrfK = 60.0

// fuseFramings RRF-merges the per-framing recall result lists into one seed:
// score(session) = Σ 1/(k + rank+1) across framings, ordered descending; the
// representative row per session is its highest-confidence hit (conf reported
// is that session's strongest framing). Knowledge and warming come from the
// primary framing. This is the folded seek.py — the "widen across phrasings"
// move, now recall over N framings.
func fuseFramings(outs []*search.Output) *search.Output {
	rrf := map[string]float64{}
	best := map[string]search.Result{}
	seen := map[string]bool{}
	var order []string // first-seen order, for stable tie-breaking

	for _, o := range outs {
		for rank, r := range o.Results {
			sid := r.SessionID
			if sid == "" {
				sid = r.Sid
			}
			if !seen[sid] {
				seen[sid] = true
				order = append(order, sid)
				best[sid] = r
			} else if r.Confidence > best[sid].Confidence {
				best[sid] = r
			}
			rrf[sid] += 1.0 / (rrfK + float64(rank) + 1.0)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return rrf[order[i]] > rrf[order[j]] })

	fused := &search.Output{
		Knowledge: outs[0].Knowledge,
		Query:     outs[0].Query,
		Filters:   outs[0].Filters,
		Mode:      outs[0].Mode,
	}
	for _, sid := range order {
		fused.Results = append(fused.Results, best[sid])
	}
	fused.Total = len(fused.Results)
	for _, o := range outs {
		if o.Semantic != nil {
			fused.Semantic = o.Semantic
			break
		}
	}
	return fused
}

// maxFramings bounds how many query reformulations one recall fuses over,
// capping the per-recall search cost. The original query is always first.
const maxFramings = 4

// framingStopwords are dropped when building the keyword-only reformulation —
// general English function words, not corpus-tuned terms.
var framingStopwords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`a an the of to in on at for and or but if is are was were be been being do does did have has had i you he she it we they me my your his her their our this that these those what when where who whom which why how with about from into over under again then once will would can could should may might must not no yes as by so than too very just also there here them us`) {
		m[w] = true
	}
	return m
}()

// framingTemporalCues trigger the time-emphasis reformulation.
var framingTemporalCues = map[string]bool{
	"when": true, "before": true, "after": true, "first": true, "last": true,
	"date": true, "day": true, "month": true, "year": true, "recently": true,
	"ago": true, "earliest": true, "latest": true,
}

var (
	framingWordRE   = regexp.MustCompile(`[A-Za-z0-9']+`)
	framingClauseRE = regexp.MustCompile(`(?i)\b(?:and|or|but|then)\b|[,;?]`)
)

// deriveFramings returns the original query plus a bounded set of deterministic
// reformulations to RRF-fuse over: a keyword-only variant (stopwords dropped),
// up to two clause splits (on conjunctions/punctuation), and — when the query
// carries a temporal cue — a time-emphasis variant. Case-insensitively
// de-duplicated and capped. A query that yields nothing new returns just
// [query], so recall stays a single search. Rules are general linguistics (no
// corpus tuning); mirrors the harness-validated multi-lookup variants.
func deriveFramings(query string) []string {
	q := strings.TrimSpace(query)
	if q == "" {
		return []string{query}
	}
	out := []string{q}
	kw := framingKeywords(q)
	if kw != "" && !strings.EqualFold(kw, q) {
		out = append(out, kw)
	}
	clauses := framingClauses(q)
	if len(clauses) > 2 {
		clauses = clauses[:2]
	}
	out = append(out, clauses...)
	if kw != "" && framingHasTemporalCue(q) {
		out = append(out, kw+" date time when")
	}
	return dedupeFramings(out)
}

// framingKeywords drops stopwords and short tokens, leaving the content words.
func framingKeywords(q string) string {
	var kept []string
	for _, t := range framingWordRE.FindAllString(strings.ToLower(q), -1) {
		if len(t) > 2 && !framingStopwords[t] {
			kept = append(kept, t)
		}
	}
	return strings.Join(kept, " ")
}

// framingClauses splits on conjunctions/punctuation, keeping parts of ≥3 words.
func framingClauses(q string) []string {
	var out []string
	for _, part := range framingClauseRE.Split(q, -1) {
		part = strings.TrimSpace(part)
		if len(strings.Fields(part)) >= 3 {
			out = append(out, part)
		}
	}
	return out
}

func framingHasTemporalCue(q string) bool {
	for _, w := range strings.Fields(strings.ToLower(q)) {
		if framingTemporalCues[w] {
			return true
		}
	}
	return false
}

// dedupeFramings drops empty/duplicate (case-insensitive) framings and caps the
// count at maxFramings, preserving first-seen order (original query first).
func dedupeFramings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		key := strings.ToLower(v)
		if v == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
		if len(out) >= maxFramings {
			break
		}
	}
	return out
}

// runRecall opens (and, if empty, rebuilds) the index DB, runs the search, and
// prints the result as JSON. The ranking/grouping engine lives in the search
// package; this function is the command-side orchestration around it.
//
// jsonCompact selects compact single-line JSON (for machine consumers) over the
// default pretty output. It is the stable structured-output opt-in for when the
// default recall output becomes a compact text digest
// (docs/design/skill-into-command.md §2.0) — machine consumers adopt --json now
// and are unaffected by that flip.
func runRecall(cmd *cobra.Command, gitRoot string, filters search.Filters, jsonCompact bool) error {
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return fmt.Errorf("open index db: %w", err)
	}
	defer indexDB.Close()

	// The index may predate optional harness-metadata columns; migrations
	// are additive and idempotent.
	if err := db.MigrateIndexSchema(indexDB); err != nil {
		return fmt.Errorf("migrate index db: %w", err)
	}

	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Auto-rebuild if the index is empty.
	if !db.IsIndexPopulated(indexDB) {
		fmt.Fprintln(cmd.ErrOrStderr(), "index not built, rebuilding...")
		indexDB.Close()
		if err := runIndex(cmd, gitRoot); err != nil {
			return err
		}
		indexDB, err = db.OpenIndex(gitRoot)
		if err != nil {
			return fmt.Errorf("reopen index db: %w", err)
		}
		defer indexDB.Close()
		if err := db.LoadFTSExtension(indexDB); err != nil {
			return fmt.Errorf("reload fts extension: %w", err)
		}
	}

	// Keep the knowledge layer fresh with HEAD. Watermark-gated: the steady
	// state (no new commits since the last refresh) costs one rev-parse; a
	// moved HEAD re-chunks only prose files whose blobs changed. Best-effort —
	// recall proceeds on a stale or absent layer.
	if err := refreshKnowledge(nil, indexDB, gitRoot); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: knowledge refresh failed: %v\n", err)
	}

	// Recall tuning + embedding backend come from .rekal/config.json. A bad
	// config falls back to defaults with a warning — recall must keep working.
	cfg, err := readMergedConfig(gitRoot)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: config unreadable, using defaults: %v\n", err)
		cfg = Config{}
	}
	weights, err := cfg.Weights.resolve()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: %v — using default weights\n", err)
		weights = search.DefaultWeights()
	}
	// Query embedder must speak the same model the index was built with.
	// When embedding is configured, use that HTTP backend — never fall back
	// to embedded nomic on resolve failure (that silently mismatches a
	// Cohere/HTTP-built index and disables the neural layer with no reason).
	var qe search.QueryEmbedder
	if cfg.Embedding != nil {
		if ec, eerr := cfg.Embedding.resolve(); eerr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: %v — semantic query embedder unavailable\n", eerr)
		} else {
			qe = embedhttp.New(ec)
		}
	}
	if embedModel, ok, _ := db.ReadIndexState(indexDB, "embed_model"); ok && embedModel != "" {
		switch {
		case qe == nil && embedModel != "nomic-v1.5":
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: index embed_model is %q but no embedding config is set — semantic layer will skip\n", embedModel)
		case qe != nil && qe.ModelName() != embedModel:
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: query embedder model %q != index embed_model %q — semantic layer may skip\n", qe.ModelName(), embedModel)
		}
	}

	// Scoring lineage is global-only and off by default. Failures opening the
	// sink warn and continue — diagnostics must never break recall.
	if lin, closer, lerr := cfg.ScoringLineage.openLineage(cmd.ErrOrStderr()); lerr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: scoring_lineage: %v — lineage disabled\n", lerr)
	} else {
		if closer != nil {
			defer closer.Close() //nolint:errcheck
		}
		filters.Lineage = lin
	}

	// Auto-widening recall: derive a small bounded set of deterministic
	// reformulations of the query (keyword-only, clause splits, a temporal
	// variant) and RRF-fuse their result lists into one seed — the folded
	// multi-lookup. The original query is always framing v0, so fusion only
	// adds/reorders hits, never drops them; a query that yields no reformulation
	// runs as a single search, byte-identical to a plain recall.
	runOne := func(q string) (search.Output, error) {
		f := filters
		f.Query = q
		f.TextQuery = strings.TrimSpace(q) != ""
		return search.Run(indexDB, f, gitRoot, weights, qe)
	}
	framings := deriveFramings(filters.Query)
	var out search.Output
	if len(framings) <= 1 {
		var oerr error
		out, oerr = runOne(filters.Query)
		if oerr != nil {
			return oerr
		}
	} else {
		outs := make([]*search.Output, 0, len(framings))
		for _, q := range framings {
			o, oerr := runOne(q)
			if oerr != nil {
				return oerr
			}
			outs = append(outs, &o)
		}
		out = *fuseFramings(outs)
	}

	// L1 recall citation graph. Read the reach hint FIRST (the state before
	// this call, so a session's own recall never inflates the number shown
	// now), attach it to the surfaced seeds, THEN spool this recall's edges for
	// the checkpoint drain. Both are best-effort: a graph failure must never
	// break recall.
	attachReach(indexDB, out.Results)
	logRecallEdges(gitRoot, filters.Query, out.Results)

	// Default output is the agent-facing seed digest (the in-binary route.py):
	// INJECT/KNOWLEDGE/SILENCE + per-seed conf=. Exit 1 on SILENCE mirrors
	// route.py so the same gating holds. --json gives raw structured results.
	if !jsonCompact {
		text, code := formatDigest(&out)
		fmt.Fprint(cmd.OutOrStdout(), text)
		if filters.Lineage != nil {
			filters.Lineage.FlushResult(len(text))
		}
		if code == 1 {
			return NewSilentError(fmt.Errorf("silence"))
		}
		return nil
	}

	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	// Flush the staged lineage "result" event with the agent-facing payload
	// size. No-op when lineage is off or nothing was staged (filter mode).
	if filters.Lineage != nil {
		filters.Lineage.FlushResult(len(data))
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
