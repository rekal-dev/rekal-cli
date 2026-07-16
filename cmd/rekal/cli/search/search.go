package search

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
)

const (
	defaultSnippetSize = 300
	defaultLimit       = 20

	// Layer weights and metadata-signal multipliers live in Weights
	// (weights.go) — configurable via .rekal/config.json, applied at query
	// time only.

	// conversationChildBudget caps how many folded transcripts are nested
	// under one collapsed conversation result, so a single large workflow
	// cannot occupy the whole top-k result budget.
	conversationChildBudget = 3
)

// Filters holds the search parameters for the recall command.
type Filters struct {
	Query  string
	File   string // regex
	Commit string // SHA prefix
	Author string // email
	Actor  string // "human" | "agent"
	Limit  int

	// Explain enriches results with per-layer scores (Layers) and
	// query-time related-session joins (Related). Off by default so the
	// standard output stays thin; benchmarking and skill-driven zooming
	// turn it on.
	Explain bool

	// Lineage, when non-nil, records observe-only scoring lineage and stage
	// timings (NDJSON). Set from the global-only scoring_lineage config —
	// never a CLI flag. Nil (the default) means zero cost: no timers, no
	// events, ranking identical to a lineage-off run.
	Lineage Lineage
}

// Layers exposes each retrieval signal's normalized [0,1] score for a
// result, before weighting — the ablation view of the hybrid rank. Present
// only under --explain and only in hybrid mode.
type Layers struct {
	BM25  float64 `json:"bm25"`
	LSA   float64 `json:"lsa"`
	Nomic float64 `json:"nomic"`
	// Facet is the facet layer's normalized score (weights.facet_boost > 0,
	// the shipped default; 0 when disabled or no facet index exists).
	Facet float64 `json:"facet"`
}

// Related is a query-time join to sessions that touched the same files —
// the co-occurrence "zoom" edge, instantiated per query instead of being a
// precomputed graph. Present only under --explain.
type Related struct {
	SessionID   string `json:"session_id"`
	SharedFiles int    `json:"shared_files"`
}

// Result is a single search result for JSON output.
type Result struct {
	SessionID      string        `json:"session_id"`
	Score          float64       `json:"score"`
	Snippet        string        `json:"snippet"`
	SnippetTurnIdx int           `json:"snippet_turn_index"`
	SnippetRole    string        `json:"snippet_role"`
	Session        SessionDetail `json:"session"`

	// Children holds other matching transcripts folded under this one
	// because they share the same trunk conversation (parent_session_id
	// chain) — subagent runs, workflow steps, or other agents in the same
	// team. Nil when this result has no conversation to fold with (the
	// common case: null parent, no matching descendants), so ungrouped
	// output is byte-identical to before grouping existed. Capped at
	// conversationChildBudget.
	Children []Result `json:"children,omitempty"`

	// SummaryTurnIdx points at the session's latest compaction-summary turn
	// (role "summary"), when one exists — summaries are cumulative, so the
	// latest subsumes the rest. It is a pointer, not a payload: the summary
	// itself is 10-17KB and inlining it would spend tokens before anyone
	// knows it's wanted. The drill is one command:
	// rekal query --session <id> --role summary. Absent when the session
	// has no summary turn.
	SummaryTurnIdx *int `json:"summary_turn_index,omitempty"`

	// Layers and Related are the --explain enrichments; nil/absent without
	// the flag so default output is unchanged.
	Layers  *Layers   `json:"layers,omitempty"`
	Related []Related `json:"related,omitempty"`
}

type SessionDetail struct {
	Author     string   `json:"author"`
	Actor      string   `json:"actor"`
	Branch     string   `json:"branch"`
	CapturedAt string   `json:"captured_at"`
	Commit     string   `json:"commit"`
	TurnCount  int      `json:"turn_count"`
	ToolCalls  int      `json:"tool_call_count"`
	Files      []string `json:"files"`

	// Optional harness metadata. Present only for sessions whose agent
	// harness has the concept (Claude Code subagents/teams/workflows);
	// omitted for everything else. Grouping/drill-down data for the agent —
	// deliberately not a CLI filter (see docs/agent-metadata.md).
	AgentID         string `json:"agent_id,omitempty"`
	TeamName        string `json:"team_name,omitempty"`
	WorkflowName    string `json:"workflow_name,omitempty"`
	ParentSessionID string `json:"parent_session_id,omitempty"`

	// Optional Task subagent meta.json sidecar fields — omitted for
	// sessions that aren't a subagent transcript (see
	// cmd/rekal/cli/session/claude.go's subagentMeta doc).
	AgentType   string `json:"agent_type,omitempty"`
	Description string `json:"description,omitempty"`
	SpawnDepth  int    `json:"spawn_depth,omitempty"`

	// Origin labels a cross-repo local import (rekal index --include*):
	// "repo:/path" or "shell:/path" for the working directory the session was
	// recorded in. Omitted (empty) for this repo's own sessions and synced
	// teammate sessions.
	Origin string `json:"origin,omitempty"`
}

type Output struct {
	Results []Result          `json:"results"`
	Query   string            `json:"query"`
	Filters map[string]string `json:"filters"`
	Mode    string            `json:"mode"`
	Total   int               `json:"total"`
}

// bm25Hit represents a BM25 match from the FTS index.
type bm25Hit struct {
	turnID    string
	sessionID string
	turnIndex int
	role      string
	content   string
	score     float64
}

func hybridSearch(indexDB *sql.DB, filters Filters, limit int, gitRoot string, w Weights, qe QueryEmbedder) ([]Result, error) {
	lin := filters.Lineage
	on := lineageOn(lin)
	var timings map[string]int64
	var skipped map[string]string
	var totalStart time.Time
	startStage := func() time.Time {
		if !on {
			return time.Time{}
		}
		return time.Now()
	}
	stage := func(name string, start time.Time) {
		if on {
			timings[name] = time.Since(start).Milliseconds()
		}
	}
	if on {
		timings = make(map[string]int64, 8)
		skipped = make(map[string]string)
		totalStart = time.Now()
	}

	// Step 1: BM25 search.
	tBM25 := startStage()
	bm25Hits, err := bm25Search(indexDB, filters.Query)
	stage("bm25", tBM25)
	if err != nil {
		return nil, fmt.Errorf("bm25 search: %w", err)
	}

	// Step 2: LSA search.
	tLSA := startStage()
	lsaScores, err := lsaSearch(indexDB, filters.Query)
	stage("lsa", tLSA)
	if err != nil {
		// LSA failure is non-fatal — fall back to BM25 only.
		lsaScores = nil
		if on {
			skipped["lsa"] = err.Error()
		}
	}

	// Step 3: Nomic deep semantic search (non-fatal).
	tNomic := startStage()
	nomicScores, nomicErr := nomicSearch(indexDB, filters.Query, gitRoot, qe)
	stage("nomic", tNomic)
	if nomicErr != nil && on {
		skipped["nomic"] = nomicErr.Error()
	}

	// Step 4: Facet layer — BM25 over per-session facet documents (tool
	// paths + command prefixes + steering text). Config-gated: at an
	// explicit FacetBoost of 0 the search never runs and ranking is
	// byte-identical to the pre-facet engine; without a facet FTS index
	// (older index.db, no facet material) the layer fails soft to nil.
	var facetScores map[string]float64
	tFacet := startStage()
	if w.FacetBoost > 0 {
		facetScores = facetSearch(indexDB, filters.Query)
		if on && facetScores == nil {
			skipped["facet"] = "no facet FTS index or empty"
		}
	} else if on {
		skipped["facet"] = "facet_boost=0"
	}
	stage("facet", tFacet)

	// Step 5: Group by session, pick best turn per session.
	tCombine := startStage()
	sessions := make(map[string]*sessionHit)

	for _, hit := range bm25Hits {
		sh, ok := sessions[hit.sessionID]
		if !ok {
			sh = &sessionHit{}
			sessions[hit.sessionID] = sh
		}
		// Steering turns (queue-operation captures) are the highest-intent
		// text in the corpus — boost them so they win the best-turn slot.
		// Summary turns (compaction distillations) are the densest — boosted
		// too, but below steering: machine text never outranks human intent
		// at equal relevance.
		weighted := hit.score
		boost := ""
		switch hit.role {
		case "human_steering":
			weighted *= w.SteeringBoost
			boost = "steering"
		case "summary":
			weighted *= w.SummaryBoost
			boost = "summary"
		}
		if weighted > sh.bm25Max {
			sh.bm25Max = weighted
			sh.bm25Raw = hit.score
			sh.bm25Boost = boost
			sh.bestHit = hit
		}
	}

	// Normalize BM25 scores to [0,1].
	var maxBM25 float64
	for _, sh := range sessions {
		if sh.bm25Max > maxBM25 {
			maxBM25 = sh.bm25Max
		}
	}

	// Add LSA scores.
	for sid, score := range lsaScores {
		sh, ok := sessions[sid]
		if !ok {
			// Pure semantic hit — need to fetch a snippet.
			sh = &sessionHit{}
			sessions[sid] = sh
		}
		sh.lsaScore = score
	}

	// Normalize LSA scores to [0,1].
	var maxLSA float64
	for _, sh := range sessions {
		if sh.lsaScore > maxLSA {
			maxLSA = sh.lsaScore
		}
	}

	// Add nomic scores.
	for sid, score := range nomicScores {
		sh, ok := sessions[sid]
		if !ok {
			sh = &sessionHit{}
			sessions[sid] = sh
		}
		sh.nomicScore = score
	}

	// Normalize nomic scores to [0,1].
	var maxNomic float64
	for _, sh := range sessions {
		if sh.nomicScore > maxNomic {
			maxNomic = sh.nomicScore
		}
	}

	// Add facet scores (empty map unless the layer is enabled).
	for sid, score := range facetScores {
		sh, ok := sessions[sid]
		if !ok {
			sh = &sessionHit{}
			sessions[sid] = sh
		}
		sh.facetScore = score
	}

	// Normalize facet scores to [0,1].
	var maxFacet float64
	for _, sh := range sessions {
		if sh.facetScore > maxFacet {
			maxFacet = sh.facetScore
		}
	}

	// Look up parent_session_id for every candidate session, once, so the
	// subagent down-weight and conversation grouping below don't re-query
	// per-session. Sessions without the concept (or without a parent) map
	// to "" — same treatment as a top-level trunk session.
	candidateIDs := make([]string, 0, len(sessions))
	for sid := range sessions {
		candidateIDs = append(candidateIDs, sid)
	}
	parentIDs, err := loadParentIDs(indexDB, candidateIDs)
	if err != nil {
		parentIDs = nil // non-fatal — falls back to no down-weighting/grouping
	}

	// Compute hybrid scores — 3-way when nomic available, 2-way fallback.
	useNomic := len(nomicScores) > 0
	bm25W3, lsaW3, nomicW3 := w.layers3()
	bm25W2, lsaW2 := w.layers2()
	var scoredResults []scored
	for sid, sh := range sessions {
		bm25Norm := 0.0
		if maxBM25 > 0 {
			bm25Norm = sh.bm25Max / maxBM25
		}
		lsaNorm := 0.0
		if maxLSA > 0 {
			lsaNorm = sh.lsaScore / maxLSA
		}
		nomicNorm := 0.0
		if useNomic && maxNomic > 0 {
			nomicNorm = sh.nomicScore / maxNomic
		}
		facetNorm := 0.0
		if maxFacet > 0 {
			facetNorm = sh.facetScore / maxFacet
		}

		var hybrid float64
		var bm25C, lsaC, nomicC float64
		if useNomic {
			bm25C, lsaC, nomicC = bm25W3*bm25Norm, lsaW3*lsaNorm, nomicW3*nomicNorm
			hybrid = bm25C + lsaC + nomicC
		} else {
			bm25C, lsaC = bm25W2*bm25Norm, lsaW2*lsaNorm
			hybrid = bm25C + lsaC
		}
		// Facet layer: additive fourth term, applied before the subagent
		// discount (hybrid += facet_boost * facetNorm). facetNorm is 0 for
		// every session when the layer is disabled or has no index, so this
		// is a no-op in those cases.
		facetC := w.FacetBoost * facetNorm
		hybrid += facetC
		hybridPreSub := hybrid
		// Subagent/workflow transcripts (non-null parent) are discounted
		// relative to trunk turns of equal relevance.
		parent := ""
		if parentIDs != nil {
			parent = parentIDs[sid]
		}
		if parent != "" {
			hybrid *= w.SubagentDownweight
		}
		sc := scored{sessionID: sid, score: hybrid, hit: sh}
		if filters.Explain {
			sc.layers = &Layers{BM25: round2(bm25Norm), LSA: round2(lsaNorm), Nomic: round2(nomicNorm), Facet: round2(facetNorm)}
		}
		if on {
			sc.lineage = &candidateLineage{
				bm25Norm:     bm25Norm,
				lsaNorm:      lsaNorm,
				nomicNorm:    nomicNorm,
				facetNorm:    facetNorm,
				bm25C:        bm25C,
				lsaC:         lsaC,
				nomicC:       nomicC,
				facetC:       facetC,
				hybridPreSub: hybridPreSub,
				parent:       parent,
			}
		}
		scoredResults = append(scoredResults, sc)
	}

	// Sort by score descending.
	sortScored(scoredResults)
	stage("combine", tCombine)

	if on {
		emitCandidateLineage(lin, scoredResults, w, useNomic)
	}

	// Build more raw candidates than the requested limit: grouping folds
	// several of them into one conversation result, so the pre-group pool
	// must be wider than the post-group output for `limit` to still mean
	// "limit conversations" rather than "limit raw hits".
	candidatePool := (limit + 1) * (conversationChildBudget + 1)

	// Apply filters and build candidate results.
	tBuild := startStage()
	results, err := buildResults(indexDB, scoredResults, filters, candidatePool)
	stage("build", tBuild)
	if err != nil {
		return nil, err
	}

	// Fold subagent/workflow hits under their trunk conversation — one
	// result per conversation, capped to `limit`.
	tGroup := startStage()
	grouped := groupByConversation(indexDB, results, parentIDs, limit)
	stage("group", tGroup)

	if on {
		timings["total"] = time.Since(totalStart).Milliseconds()
		norm := LineageNormWeights{}
		if useNomic {
			norm.BM25, norm.LSA, norm.Nomic = bm25W3, lsaW3, nomicW3
		} else {
			norm.BM25, norm.LSA = bm25W2, lsaW2
		}
		embedModel := ""
		if qe != nil {
			embedModel = qe.ModelName()
		} else if useNomic {
			embedModel = nomic.ModelName
		}
		var skipOut map[string]string
		if len(skipped) > 0 {
			skipOut = skipped
		}
		lin.Emit(LineageQuery{
			Type:    "query",
			TS:      time.Now().UTC(),
			Query:   filters.Query,
			Mode:    "hybrid",
			Filters: map[string]string{"file": filters.File, "actor": filters.Actor, "commit": filters.Commit, "author": filters.Author},
			Weights: LineageWeights{
				BM25: w.BM25, LSA: w.LSA, Nomic: w.Nomic,
				SteeringBoost: w.SteeringBoost, SummaryBoost: w.SummaryBoost,
				SubagentDownweight: w.SubagentDownweight, FacetBoost: w.FacetBoost,
			},
			WeightsNormalized: norm,
			UseNomic:          useNomic,
			EmbedderModel:     embedModel,
			TimingsMS:         timings,
			Counts: map[string]int{
				"bm25_hits":      len(bm25Hits),
				"lsa_sessions":   len(lsaScores),
				"nomic_sessions": len(nomicScores),
				"facet_hits":     len(facetScores),
				"candidates":     len(scoredResults),
				"after_filter":   len(results),
				"returned":       len(grouped),
			},
			Skipped: skipOut,
		})
	}

	return grouped, nil
}

// candidateLineage is the per-session math snapshot retained only when
// lineage logging is on — kept off the hot path when Lineage is nil.
type candidateLineage struct {
	bm25Norm, lsaNorm, nomicNorm, facetNorm float64
	bm25C, lsaC, nomicC, facetC             float64
	hybridPreSub                            float64
	parent                                  string
}

func emitCandidateLineage(lin Lineage, scoredResults []scored, w Weights, useNomic bool) {
	n := lin.MaxCandidates()
	if n <= 0 {
		n = 50
	}
	if n > len(scoredResults) {
		n = len(scoredResults)
	}
	for i := 0; i < n; i++ {
		sc := scoredResults[i]
		cl := sc.lineage
		if cl == nil || sc.hit == nil {
			continue
		}
		sh := sc.hit
		ev := LineageCandidate{
			Type:         "candidate",
			SessionID:    sc.sessionID,
			RankPreGroup: i + 1,
			BM25:         LineageLayer{Raw: round4(sh.bm25Raw), Norm: round4(cl.bm25Norm)},
			LSA:          LineageLayer{Raw: round4(sh.lsaScore), Norm: round4(cl.lsaNorm)},
			Nomic:        LineageLayer{Raw: round4(sh.nomicScore), Norm: round4(cl.nomicNorm)},
			Facet:        LineageLayer{Raw: round4(sh.facetScore), Norm: round4(cl.facetNorm)},
			Contrib: map[string]float64{
				"bm25":  round4(cl.bm25C),
				"lsa":   round4(cl.lsaC),
				"facet": round4(cl.facetC),
			},
			HybridPreSub: round4(cl.hybridPreSub),
			Score:        round4(sc.score),
		}
		if useNomic {
			ev.Contrib["nomic"] = round4(cl.nomicC)
		}
		if sh.bestHit.sessionID != "" || sh.bm25Max > 0 {
			ev.BestTurn = &LineageBestTurn{
				Index: sh.bestHit.turnIndex,
				Role:  sh.bestHit.role,
				Boost: sh.bm25Boost,
			}
		}
		if cl.parent != "" {
			ev.Subagent = &LineageSubagent{Parent: cl.parent, Multiplier: w.SubagentDownweight}
		}
		lin.Emit(ev)
	}
}

// round4 keeps lineage numbers readable without changing ranking math.
func round4(f float64) float64 { return math.Round(f*10000) / 10000 }

func filterSearch(indexDB *sql.DB, filters Filters, limit int) ([]Result, error) {
	// Build WHERE clause from filters.
	where, args := buildFilterWhere(filters)

	query := "SELECT " + sessionFacetCols + " FROM session_facets"
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY captured_at DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := indexDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("filter query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var results []Result
	for rows.Next() {
		var sf sessionFacetRow
		if err := sf.scan(rows); err != nil {
			return nil, fmt.Errorf("scan facet: %w", err)
		}

		files, _ := querySessionFiles(indexDB, sf.sessionID)
		snippet, turnIdx, role := firstTurnSnippet(indexDB, sf.sessionID)

		results = append(results, Result{
			SessionID:      sf.sessionID,
			Score:          0,
			Snippet:        snippet,
			SnippetTurnIdx: turnIdx,
			SnippetRole:    role,
			Session:        sf.detail(files),
		})
	}
	return results, rows.Err()
}

// sessionFacetCols is the column list matching sessionFacetRow.scan. The
// harness-metadata columns are nullable — NULL for agents without the concept.
const sessionFacetCols = "session_id, user_email, git_branch, actor_type, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha, agent_id, team_name, workflow_name, parent_session_id, agent_type, description, spawn_depth, origin"

type sessionFacetRow struct {
	sessionID     string
	email         sql.NullString
	branch        sql.NullString
	actorType     string
	capturedAt    string
	turnCount     int
	toolCallCount int
	fileCount     int
	checkpointID  sql.NullString
	gitSHA        sql.NullString

	// Optional harness metadata — NULL for agents without the concept.
	agentID         sql.NullString
	teamName        sql.NullString
	workflowName    sql.NullString
	parentSessionID sql.NullString

	// Optional Task subagent meta.json sidecar fields — NULL for sessions
	// that aren't a subagent transcript.
	agentType   sql.NullString
	description sql.NullString
	spawnDepth  sql.NullInt64

	// origin — NULL for this repo's own sessions and synced teammate data;
	// set only for cross-repo local imports.
	origin sql.NullString
}

// scanner abstracts *sql.Row and *sql.Rows for sessionFacetRow.scan.
type scanner interface {
	Scan(dest ...interface{}) error
}

func (sf *sessionFacetRow) scan(s scanner) error {
	return s.Scan(&sf.sessionID, &sf.email, &sf.branch, &sf.actorType, &sf.capturedAt,
		&sf.turnCount, &sf.toolCallCount, &sf.fileCount, &sf.checkpointID, &sf.gitSHA,
		&sf.agentID, &sf.teamName, &sf.workflowName, &sf.parentSessionID,
		&sf.agentType, &sf.description, &sf.spawnDepth, &sf.origin)
}

// detail builds the JSON session detail; optional metadata fields are only
// set when present so they are omitted from output for agents without them.
func (sf *sessionFacetRow) detail(files []string) SessionDetail {
	return SessionDetail{
		Author:          nullStr(sf.email),
		Actor:           sf.actorType,
		Branch:          nullStr(sf.branch),
		CapturedAt:      sf.capturedAt,
		Commit:          nullStr(sf.gitSHA),
		TurnCount:       sf.turnCount,
		ToolCalls:       sf.toolCallCount,
		Files:           files,
		AgentID:         nullStr(sf.agentID),
		TeamName:        nullStr(sf.teamName),
		WorkflowName:    nullStr(sf.workflowName),
		ParentSessionID: nullStr(sf.parentSessionID),
		AgentType:       nullStr(sf.agentType),
		Description:     nullStr(sf.description),
		SpawnDepth:      int(sf.spawnDepth.Int64),
		Origin:          nullStr(sf.origin),
	}
}

func buildFilterWhere(filters Filters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filters.Actor != "" {
		conditions = append(conditions, fmt.Sprintf("actor_type = $%d", idx))
		args = append(args, filters.Actor)
		idx++
	}
	if filters.Author != "" {
		conditions = append(conditions, fmt.Sprintf("user_email = $%d", idx))
		args = append(args, filters.Author)
		idx++
	}
	if filters.Commit != "" {
		conditions = append(conditions, fmt.Sprintf("git_sha LIKE $%d", idx))
		args = append(args, filters.Commit+"%")
		idx++
	}
	if filters.File != "" {
		conditions = append(conditions, fmt.Sprintf("session_id IN (SELECT DISTINCT session_id FROM files_index WHERE regexp_matches(file_path, $%d))", idx))
		args = append(args, filters.File)
	}

	return strings.Join(conditions, " AND "), args
}

func bm25Search(indexDB *sql.DB, query string) ([]bm25Hit, error) {
	// Check if FTS index exists (it won't if there are no turns).
	var count int
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&count); err != nil || count == 0 {
		return nil, nil
	}

	rows, err := indexDB.Query(`
		SELECT ft.id, ft.session_id, ft.turn_index, ft.role, ft.content,
		       fts_main_turns_ft.match_bm25(ft.id, $1) AS score
		FROM turns_ft ft
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT 200
	`, query)
	if err != nil {
		// FTS index may not exist — return empty gracefully.
		return nil, nil
	}
	defer rows.Close() //nolint:errcheck

	var hits []bm25Hit
	for rows.Next() {
		var h bm25Hit
		if err := rows.Scan(&h.turnID, &h.sessionID, &h.turnIndex, &h.role, &h.content, &h.score); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// facetSearch scores sessions by BM25 over their facet document
// (session_facets.facet_text — distinct tool paths + command prefixes +
// steering text; db.PopulateFacetText). Only called when the layer is
// enabled (weights.facet_boost > 0). Non-fatal by construction: when the
// facet FTS index does not exist (older index.db, or a corpus with no
// facet material — db.CreateFacetFTSIndex is guarded), the query errors
// and the layer is skipped.
func facetSearch(indexDB *sql.DB, query string) map[string]float64 {
	rows, err := indexDB.Query(`
		SELECT sf.session_id,
		       fts_main_session_facets.match_bm25(sf.session_id, $1) AS score
		FROM session_facets sf
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT 200
	`, query)
	if err != nil {
		return nil // facet FTS index absent — layer off
	}
	defer rows.Close() //nolint:errcheck

	scores := make(map[string]float64)
	for rows.Next() {
		var sid string
		var score float64
		if err := rows.Scan(&sid, &score); err != nil {
			return nil
		}
		scores[sid] = score
	}
	if rows.Err() != nil {
		return nil
	}
	return scores
}

func lsaSearch(indexDB *sql.DB, query string) (map[string]float64, error) {
	// Load LSA embeddings only.
	embeddings, err := db.QueryEmbeddings(indexDB, "lsa-v1")
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	// We need the LSA model to project the query. Prefer the projection
	// state cached at index/sync time (storeLSAProjection) — rebuilding the
	// whole model from raw content on every single recall call means
	// re-tokenizing the entire corpus and re-running SVD per query, which
	// gets slower as the corpus grows. Fall back to a full rebuild only for
	// an index.db that predates this cache or never got one written (e.g.
	// LSA build failed at index time but embeddings still exist from an
	// earlier run).
	model, err := loadLSAProjection(indexDB)
	if err != nil {
		model = nil // non-fatal — fall back to rebuilding
	}
	if model == nil {
		sessionContent, cErr := db.QuerySessionContent(indexDB)
		if cErr != nil {
			return nil, cErr
		}
		model, err = lsa.Build(sessionContent, lsa.DefaultDimension)
		if err != nil || model == nil {
			return nil, err
		}
	}

	queryVec := model.Embed(query)

	scores := make(map[string]float64)
	for sid, emb := range embeddings {
		sim := lsa.CosineSimilarity(queryVec, emb)
		if sim > 0 {
			scores[sid] = sim
		}
	}
	return scores, nil
}

// QueryEmbedder embeds recall queries into the same vector space the index's
// session embeddings were generated in. Implemented by the embedded nomic
// client and the HTTP backend (cli wires the right one from config).
type QueryEmbedder interface {
	// ModelName keys the stored vectors this embedder is compatible with.
	ModelName() string
	EmbedQuery(text string) ([]float64, error)
	Close()
}

// nomicSearch computes deep semantic similarity against the index's stored
// session vectors. Non-fatal: returns nil on any failure or when no backend
// is available.
//
// The embedder must match the model the index was built with — vectors from
// different models live in incompatible spaces, so on a mismatch the layer is
// skipped (scores nil → hybrid falls back to 2-way weights) rather than
// comparing garbage. qe == nil selects the embedded nomic backend.
func nomicSearch(indexDB *sql.DB, query string, gitRoot string, qe QueryEmbedder) (map[string]float64, error) {
	if qe == nil {
		if !nomic.Supported() {
			return nil, nil
		}
		client, err := nomic.NewClient(gitRoot)
		if err != nil {
			return nil, err
		}
		qe = nomicQueryEmbedder{client}
	}
	defer qe.Close()

	// Load stored vectors for this embedder's model; none (e.g. the index was
	// built with a different backend) means the layer is skipped.
	embeddings, err := db.QueryEmbeddings(indexDB, qe.ModelName())
	if err != nil || len(embeddings) == 0 {
		return nil, err
	}

	queryVec, err := qe.EmbedQuery(query)
	if err != nil {
		return nil, err
	}

	scores := make(map[string]float64)
	for sid, emb := range embeddings {
		sim := lsa.CosineSimilarity(queryVec, emb)
		if sim > 0 {
			scores[sid] = sim
		}
	}
	return scores, nil
}

// nomicQueryEmbedder adapts the embedded nomic client to QueryEmbedder.
type nomicQueryEmbedder struct{ c *nomic.Client }

func (n nomicQueryEmbedder) ModelName() string                         { return nomic.ModelName }
func (n nomicQueryEmbedder) EmbedQuery(text string) ([]float64, error) { return n.c.EmbedQuery(text) }
func (n nomicQueryEmbedder) Close()                                    { n.c.Close() }

func buildResults(indexDB *sql.DB, scored []scored, filters Filters, limit int) ([]Result, error) {
	// Compile file regex if present.
	var fileRe *regexp.Regexp
	if filters.File != "" {
		var err error
		fileRe, err = regexp.Compile(filters.File)
		if err != nil {
			return nil, fmt.Errorf("invalid file regex: %w", err)
		}
	}

	// Batch-load facets and files for every candidate up front — candidatePool
	// (hybridSearch) is deliberately wider than limit so grouping has raw
	// material to fold, which used to mean one facets query and one files
	// query per candidate here. loadParentIDs already does this batched;
	// this mirrors that pattern for the other two per-row lookups.
	candidateIDs := make([]string, len(scored))
	for i, s := range scored {
		candidateIDs[i] = s.sessionID
	}
	facets, err := loadSessionFacetsBatch(indexDB, candidateIDs)
	if err != nil {
		return nil, fmt.Errorf("load session facets: %w", err)
	}
	filesBySession, err := loadSessionFilesBatch(indexDB, candidateIDs)
	if err != nil {
		filesBySession = nil // non-fatal — file filter/output just comes back empty
	}

	var results []Result
	for _, s := range scored {
		if len(results) >= limit {
			break
		}

		sf, ok := facets[s.sessionID]
		if !ok {
			continue // session not in facets (shouldn't happen)
		}

		// Apply filters.
		if filters.Actor != "" && sf.actorType != filters.Actor {
			continue
		}
		if filters.Author != "" && nullStr(sf.email) != filters.Author {
			continue
		}
		if filters.Commit != "" && !strings.HasPrefix(nullStr(sf.gitSHA), filters.Commit) {
			continue
		}

		files := filesBySession[s.sessionID]

		if fileRe != nil {
			matched := false
			for _, f := range files {
				if fileRe.MatchString(f) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Build snippet.
		var snippet string
		var snippetIdx int
		var snippetRole string

		if s.hit != nil && s.hit.bestHit.content != "" {
			snippet = extractSnippet(s.hit.bestHit.content, filters.Query)
			snippetIdx = s.hit.bestHit.turnIndex
			snippetRole = s.hit.bestHit.role
		} else {
			snippet, snippetIdx, snippetRole = firstTurnSnippet(indexDB, s.sessionID)
		}

		results = append(results, Result{
			SessionID:      s.sessionID,
			Score:          math.Round(s.score*100) / 100,
			Snippet:        snippet,
			SnippetTurnIdx: snippetIdx,
			SnippetRole:    snippetRole,
			Session:        sf.detail(files),
			Layers:         s.layers,
		})
	}

	return results, nil
}

// attachSummaryPointers sets SummaryTurnIdx on every result (and folded
// child) whose session has a compaction-summary turn — the pointer that
// makes the cheap-overview drill discoverable without inlining a 10-17KB
// payload into recall output (progressive disclosure; see Result doc). One
// batched query for the whole result set; failures are non-fatal and simply
// leave the field absent.
func attachSummaryPointers(indexDB *sql.DB, results []Result) {
	if len(results) == 0 {
		return
	}
	var ids []string
	for _, r := range results {
		ids = append(ids, r.SessionID)
		for _, c := range r.Children {
			ids = append(ids, c.SessionID)
		}
	}
	inClause, args := sqlInClause(ids)
	rows, err := indexDB.Query(`
		SELECT session_id, MAX(turn_index)
		FROM turns_ft
		WHERE role = 'summary' AND session_id IN (`+inClause+`)
		GROUP BY session_id`, args...)
	if err != nil {
		return
	}
	defer rows.Close() //nolint:errcheck

	latest := make(map[string]int)
	for rows.Next() {
		var sid string
		var idx int
		if err := rows.Scan(&sid, &idx); err != nil {
			return
		}
		latest[sid] = idx
	}
	if rows.Err() != nil {
		return
	}

	for i := range results {
		if idx, ok := latest[results[i].SessionID]; ok {
			v := idx
			results[i].SummaryTurnIdx = &v
		}
		for j := range results[i].Children {
			if idx, ok := latest[results[i].Children[j].SessionID]; ok {
				v := idx
				results[i].Children[j].SummaryTurnIdx = &v
			}
		}
	}
}

// relatedLimit caps the co-occurrence join per result — enough edges to zoom
// along, thin enough to stay cheap on the wire.
const relatedLimit = 3

// attachRelated enriches top-level results with query-time joins to sessions
// sharing touched files (file co-occurrence, instantiated per query — no
// precomputed graph). One batched query for the whole result set; failures
// are non-fatal and simply leave Related empty.
func attachRelated(indexDB *sql.DB, results []Result) {
	if len(results) == 0 {
		return
	}
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.SessionID
	}
	inClause, args := sqlInClause(ids)
	rows, err := indexDB.Query(`
		SELECT a.session_id, b.session_id, count(DISTINCT b.file_path) AS shared
		FROM files_index a
		JOIN files_index b ON a.file_path = b.file_path AND b.session_id <> a.session_id
		WHERE a.session_id IN (`+inClause+`)
		GROUP BY a.session_id, b.session_id
		ORDER BY a.session_id, shared DESC`, args...)
	if err != nil {
		return
	}
	defer rows.Close() //nolint:errcheck

	related := make(map[string][]Related, len(results))
	for rows.Next() {
		var src, dst string
		var shared int
		if err := rows.Scan(&src, &dst, &shared); err != nil {
			return
		}
		if len(related[src]) < relatedLimit {
			related[src] = append(related[src], Related{SessionID: dst, SharedFiles: shared})
		}
	}
	if rows.Err() != nil {
		return
	}
	for i := range results {
		results[i].Related = related[results[i].SessionID]
	}
}

// sqlInClause builds a "$1,$2,...,$N" placeholder list and the matching args
// slice for an `IN (...)` clause over sessionIDs. It assumes $1 is the first
// bound parameter in the statement (nothing else precedes it).
func sqlInClause(sessionIDs []string) (string, []interface{}) {
	placeholders := make([]string, len(sessionIDs))
	args := make([]interface{}, len(sessionIDs))
	for i, sid := range sessionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sid
	}
	return strings.Join(placeholders, ","), args
}

// loadSessionFacetsBatch batch-loads session_facets rows for a set of
// candidate sessions, keyed by session_id. Sessions missing from
// session_facets (shouldn't happen) are simply absent from the map.
func loadSessionFacetsBatch(indexDB *sql.DB, sessionIDs []string) (map[string]sessionFacetRow, error) {
	result := make(map[string]sessionFacetRow, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return result, nil
	}

	inClause, args := sqlInClause(sessionIDs)
	rows, err := indexDB.Query(
		"SELECT "+sessionFacetCols+" FROM session_facets WHERE session_id IN ("+inClause+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch load session facets: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var sf sessionFacetRow
		if err := sf.scan(rows); err != nil {
			return nil, fmt.Errorf("scan session facet: %w", err)
		}
		result[sf.sessionID] = sf
	}
	return result, rows.Err()
}

// loadSessionFilesBatch batch-loads distinct file paths per session for a
// set of candidate sessions, keyed by session_id. Sessions with no files
// touched are simply absent from the map (nil slice on lookup, same as
// querySessionFiles returning no rows).
func loadSessionFilesBatch(indexDB *sql.DB, sessionIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return result, nil
	}

	inClause, args := sqlInClause(sessionIDs)
	rows, err := indexDB.Query(
		"SELECT DISTINCT session_id, file_path FROM files_index WHERE session_id IN ("+inClause+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch load session files: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var sid, path string
		if err := rows.Scan(&sid, &path); err != nil {
			return nil, fmt.Errorf("scan session file: %w", err)
		}
		result[sid] = append(result[sid], path)
	}
	return result, rows.Err()
}

// loadParentIDs batch-loads parent_session_id for a set of candidate
// sessions. Sessions with no parent (or no harness concept of one) map to
// "" — treated identically to a top-level trunk session downstream.
func loadParentIDs(indexDB *sql.DB, sessionIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return result, nil
	}

	for _, sid := range sessionIDs {
		result[sid] = "" // default: no parent, until proven otherwise
	}

	inClause, args := sqlInClause(sessionIDs)
	rows, err := indexDB.Query(
		"SELECT session_id, parent_session_id FROM session_facets WHERE session_id IN ("+inClause+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("load parent ids: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var sid string
		var parent sql.NullString
		if err := rows.Scan(&sid, &parent); err != nil {
			return nil, fmt.Errorf("scan parent id: %w", err)
		}
		result[sid] = nullStr(parent)
	}
	return result, rows.Err()
}

// resolveRoot walks the parent_session_id chain to find the trunk
// conversation for sid, memoizing lookups in cache across calls within one
// search. A session with no parent is its own root.
func resolveRoot(indexDB *sql.DB, sid string, cache map[string]string) string {
	visited := make(map[string]bool)
	cur := sid
	for {
		if visited[cur] {
			return cur // cycle guard — should not happen, but never loop forever
		}
		visited[cur] = true

		parent, ok := cache[cur]
		if !ok {
			var ns sql.NullString
			err := indexDB.QueryRow(
				"SELECT parent_session_id FROM session_facets WHERE session_id = $1", cur,
			).Scan(&ns)
			if err != nil {
				cache[cur] = ""
				return cur
			}
			parent = nullStr(ns)
			cache[cur] = parent
		}
		if parent == "" {
			return cur
		}
		cur = parent
	}
}

// groupByConversation folds subagent/workflow transcript results under their
// trunk conversation (docs/agent-metadata.md: grouping via parent_session_id,
// generic across agent types). Results is assumed sorted by score descending.
// One output entry per conversation, ordered by that conversation's best
// score, capped to limit conversations and conversationChildBudget children
// each.
func groupByConversation(indexDB *sql.DB, results []Result, parentIDs map[string]string, limit int) []Result {
	rootCache := make(map[string]string, len(parentIDs))
	for sid, parent := range parentIDs {
		rootCache[sid] = parent
	}

	type group struct {
		root    string
		members []Result
	}

	order := make([]string, 0, len(results))
	groups := make(map[string]*group, len(results))

	for _, r := range results {
		root := resolveRoot(indexDB, r.SessionID, rootCache)
		g, ok := groups[root]
		if !ok {
			g = &group{root: root}
			groups[root] = g
			order = append(order, root)
		}
		if len(g.members) < conversationChildBudget {
			g.members = append(g.members, r)
		}
	}

	out := make([]Result, 0, limit)
	for _, root := range order {
		if len(out) >= limit {
			break
		}
		g := groups[root]
		best := g.members[0]
		children := g.members[1:]

		if best.SessionID == root {
			// The trunk conversation itself is the best match — present it
			// as today, with any folded transcripts nested beneath.
			best.Children = children
			out = append(out, best)
			continue
		}

		// The best match is a subagent/workflow transcript — surface the
		// trunk conversation's identity at the top level (its content plus
		// score/snippet inherited from the best-matching descendant), and
		// fold every matching transcript, including the best one, beneath
		// it so the agent can drill straight into it.
		trunk, ok := loadSessionFacet(indexDB, root)
		if !ok {
			// Trunk not indexed (dangling parent_session_id) — fall back to
			// presenting the best match ungrouped, same as before grouping.
			best.Children = children
			out = append(out, best)
			continue
		}
		out = append(out, Result{
			SessionID:      root,
			Score:          best.Score,
			Snippet:        best.Snippet,
			SnippetTurnIdx: best.SnippetTurnIdx,
			SnippetRole:    best.SnippetRole,
			Session:        trunk,
			Children:       g.members,
		})
	}
	return out
}

// loadSessionFacet loads a session's facet row for use as a synthetic group
// header when the trunk conversation itself did not match the query.
func loadSessionFacet(indexDB *sql.DB, sessionID string) (SessionDetail, bool) {
	var sf sessionFacetRow
	row := indexDB.QueryRow("SELECT "+sessionFacetCols+" FROM session_facets WHERE session_id = $1", sessionID)
	if err := sf.scan(row); err != nil {
		return SessionDetail{}, false
	}
	files, _ := querySessionFiles(indexDB, sessionID)
	return sf.detail(files), true
}

type scored struct {
	sessionID string
	score     float64
	hit       *sessionHit
	layers    *Layers           // populated only under Filters.Explain
	lineage   *candidateLineage // populated only when Filters.Lineage is set
}

// round2 rounds to two decimals for stable, thin JSON output.
func round2(f float64) float64 { return math.Round(f*100) / 100 }

type sessionHit struct {
	bestHit    bm25Hit
	bm25Max    float64 // boosted BM25 of the winning turn
	bm25Raw    float64 // pre-boost BM25 of the winning turn (lineage)
	bm25Boost  string  // "steering" | "summary" | "" (lineage)
	lsaScore   float64
	nomicScore float64
	facetScore float64
}

func sortScored(s []scored) {
	// Sort descending by score. LSA scoring touches every session in the
	// corpus, so this list grows with history — keep it O(n log n).
	sort.Slice(s, func(i, j int) bool { return s[i].score > s[j].score })
}

func querySessionFiles(indexDB *sql.DB, sessionID string) ([]string, error) {
	rows, err := indexDB.Query("SELECT DISTINCT file_path FROM files_index WHERE session_id = $1", sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var files []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func firstTurnSnippet(indexDB *sql.DB, sessionID string) (string, int, string) {
	var content, role string
	var turnIndex int
	err := indexDB.QueryRow(
		"SELECT turn_index, role, content FROM turns_ft WHERE session_id = $1 ORDER BY turn_index LIMIT 1",
		sessionID,
	).Scan(&turnIndex, &role, &content)
	if err != nil {
		return "", 0, ""
	}
	if len(content) > defaultSnippetSize {
		content = content[:snapToRuneStart(content, defaultSnippetSize)] + "..."
	}
	return content, turnIndex, role
}

// snapToRuneStart moves i backward to the start of the UTF-8 rune containing
// it, so byte-offset snippet boundaries never split a multi-byte character.
func snapToRuneStart(s string, i int) int {
	for i > 0 && i < len(s) && !utf8.RuneStart(s[i]) {
		i--
	}
	return i
}

// extractSnippet extracts a window around the first query term match.
func extractSnippet(content, query string) string {
	if len(content) <= defaultSnippetSize {
		return content
	}

	lower := strings.ToLower(content)
	terms := lsa.Tokenize(query)

	bestPos := -1
	for _, term := range terms {
		pos := strings.Index(lower, term)
		if pos >= 0 && (bestPos < 0 || pos < bestPos) {
			bestPos = pos
		}
	}

	if bestPos < 0 {
		// No term match — take first N chars.
		return content[:snapToRuneStart(content, defaultSnippetSize)] + "..."
	}

	half := defaultSnippetSize / 2
	start := bestPos - half
	if start < 0 {
		start = 0
	}
	end := start + defaultSnippetSize
	if end > len(content) {
		end = len(content)
		start = end - defaultSnippetSize
		if start < 0 {
			start = 0
		}
	}

	// Never split a multi-byte rune. The word alignment below preserves this:
	// it only ever moves a boundary to sit next to an ASCII space.
	start = snapToRuneStart(content, start)
	end = snapToRuneStart(content, end)

	// Align to word boundaries. If the window holds one giant token (no
	// space at all), leave the boundary where it is rather than walking
	// start past end — content[start:end] must stay a valid slice.
	if start > 0 {
		aligned := start
		for aligned < end && content[aligned] != ' ' {
			aligned++
		}
		if aligned < end {
			start = aligned + 1 // skip the space
		}
	}
	if end < len(content) {
		aligned := end
		for aligned > start && content[aligned-1] != ' ' {
			aligned--
		}
		if aligned > start {
			end = aligned
		}
	}

	snippet := content[start:end]
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(content) {
		suffix = "..."
	}
	return prefix + snippet + suffix
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Run executes a recall query against an already-open, migrated, FTS-loaded
// index DB and returns the assembled Output. Hybrid ranking (BM25 + LSA +
// Nomic) is used when a text query is present; otherwise results come from
// facet filtering alone. Opening/rebuilding the index and marshaling the
// Output to JSON are the caller's responsibility (see cli.runRecall).
// Run executes a recall. w tunes ranking (zero value → defaults); qe embeds
// the query for the deep semantic layer (nil → the embedded nomic backend).
func Run(indexDB *sql.DB, filters Filters, gitRoot string, w Weights, qe QueryEmbedder) (Output, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	w = w.orDefaults()

	var results []Result
	var err error
	mode := "filter"
	if filters.Query != "" {
		mode = "hybrid"
		results, err = hybridSearch(indexDB, filters, limit, gitRoot, w, qe)
	} else {
		results, err = filterSearch(indexDB, filters, limit)
	}
	if err != nil {
		return Output{}, err
	}

	attachSummaryPointers(indexDB, results)

	if filters.Explain {
		attachRelated(indexDB, results)
	}

	return Output{
		Results: results,
		Query:   filters.Query,
		Filters: map[string]string{
			"file":   filters.File,
			"actor":  filters.Actor,
			"commit": filters.Commit,
			"author": filters.Author,
		},
		Mode:  mode,
		Total: len(results),
	}, nil
}
