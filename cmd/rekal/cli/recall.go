package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/spf13/cobra"
)

const (
	defaultSnippetSize = 300
	defaultLimit       = 20

	// 2-way weights (fallback when nomic is unavailable).
	bm25Weight2Way = 0.4
	lsaWeight2Way  = 0.6

	// 3-way weights (full hybrid with nomic).
	bm25Weight3Way  = 0.35 // Keyword precision
	lsaWeight3Way   = 0.10 // Corpus-specific co-occurrence
	nomicWeight3Way = 0.55 // Semantic understanding

	// Signal weighting over harness metadata (docs/agent-metadata.md). Simple
	// multipliers, not a re-ranking pass — applied where the existing hybrid
	// score is already assembled.
	//
	// steeringBoost favors turns captured from queue-operation/enqueue: text
	// typed by a human while an agent was already working is the highest-
	// intent signal in the corpus (SOUL.md: "agent first").
	steeringBoost = 1.3
	// subagentDownweight discounts sessions that are not the trunk of their
	// conversation (non-null parent_session_id) relative to trunk turns of
	// equal textual relevance — a subagent's internal exploration matters
	// less than what the trunk actually said or decided.
	subagentDownweight = 0.7
	// conversationChildBudget caps how many folded transcripts are nested
	// under one collapsed conversation result, so a single large workflow
	// cannot occupy the whole top-k result budget.
	conversationChildBudget = 3
)

// RecallFilters holds the search parameters for the recall command.
type RecallFilters struct {
	Query  string
	File   string // regex
	Commit string // SHA prefix
	Author string // email
	Actor  string // "human" | "agent"
	Limit  int
}

// searchResult is a single search result for JSON output.
type searchResult struct {
	SessionID      string        `json:"session_id"`
	Score          float64       `json:"score"`
	Snippet        string        `json:"snippet"`
	SnippetTurnIdx int           `json:"snippet_turn_index"`
	SnippetRole    string        `json:"snippet_role"`
	Session        sessionDetail `json:"session"`

	// Children holds other matching transcripts folded under this one
	// because they share the same trunk conversation (parent_session_id
	// chain) — subagent runs, workflow steps, or other agents in the same
	// team. Nil when this result has no conversation to fold with (the
	// common case: null parent, no matching descendants), so ungrouped
	// output is byte-identical to before grouping existed. Capped at
	// conversationChildBudget.
	Children []searchResult `json:"children,omitempty"`
}

type sessionDetail struct {
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
}

type searchOutput struct {
	Results []searchResult    `json:"results"`
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

func runRecall(cmd *cobra.Command, gitRoot string, filters RecallFilters) error {
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

	// Load FTS extension.
	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Auto-rebuild if index is empty.
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

	limit := filters.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	var results []searchResult
	mode := "filter"

	if filters.Query != "" {
		mode = "hybrid"
		results, err = hybridSearch(indexDB, filters, limit, gitRoot)
	} else {
		results, err = filterSearch(indexDB, filters, limit)
	}
	if err != nil {
		return err
	}

	output := searchOutput{
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
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func hybridSearch(indexDB *sql.DB, filters RecallFilters, limit int, gitRoot string) ([]searchResult, error) {
	// Step 1: BM25 search.
	bm25Hits, err := bm25Search(indexDB, filters.Query)
	if err != nil {
		return nil, fmt.Errorf("bm25 search: %w", err)
	}

	// Step 2: LSA search.
	lsaScores, err := lsaSearch(indexDB, filters.Query)
	if err != nil {
		// LSA failure is non-fatal — fall back to BM25 only.
		lsaScores = nil
	}

	// Step 3: Nomic deep semantic search (non-fatal).
	nomicScores, _ := nomicSearch(indexDB, filters.Query, gitRoot)

	// Step 4: Group by session, pick best turn per session.
	sessions := make(map[string]*sessionHit)

	for _, hit := range bm25Hits {
		sh, ok := sessions[hit.sessionID]
		if !ok {
			sh = &sessionHit{}
			sessions[hit.sessionID] = sh
		}
		// Steering turns (queue-operation captures) are the highest-intent
		// text in the corpus — boost them so they win the best-turn slot.
		weighted := hit.score
		if hit.role == "human_steering" {
			weighted *= steeringBoost
		}
		if weighted > sh.bm25Max {
			sh.bm25Max = weighted
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

		var hybrid float64
		if useNomic {
			nomicNorm := 0.0
			if maxNomic > 0 {
				nomicNorm = sh.nomicScore / maxNomic
			}
			hybrid = bm25Weight3Way*bm25Norm + lsaWeight3Way*lsaNorm + nomicWeight3Way*nomicNorm
		} else {
			hybrid = bm25Weight2Way*bm25Norm + lsaWeight2Way*lsaNorm
		}
		// Subagent/workflow transcripts (non-null parent) are discounted
		// relative to trunk turns of equal relevance.
		if parentIDs[sid] != "" {
			hybrid *= subagentDownweight
		}
		scoredResults = append(scoredResults, scored{sid, hybrid, sh})
	}

	// Sort by score descending.
	sortScored(scoredResults)

	// Build more raw candidates than the requested limit: grouping folds
	// several of them into one conversation result, so the pre-group pool
	// must be wider than the post-group output for `limit` to still mean
	// "limit conversations" rather than "limit raw hits".
	candidatePool := (limit + 1) * (conversationChildBudget + 1)

	// Apply filters and build candidate results.
	results, err := buildResults(indexDB, scoredResults, filters, candidatePool)
	if err != nil {
		return nil, err
	}

	// Fold subagent/workflow hits under their trunk conversation — one
	// result per conversation, capped to `limit`.
	return groupByConversation(indexDB, results, parentIDs, limit), nil
}

func filterSearch(indexDB *sql.DB, filters RecallFilters, limit int) ([]searchResult, error) {
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

	var results []searchResult
	for rows.Next() {
		var sf sessionFacetRow
		if err := sf.scan(rows); err != nil {
			return nil, fmt.Errorf("scan facet: %w", err)
		}

		files, _ := querySessionFiles(indexDB, sf.sessionID)
		snippet, turnIdx, role := firstTurnSnippet(indexDB, sf.sessionID)

		results = append(results, searchResult{
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
const sessionFacetCols = "session_id, user_email, git_branch, actor_type, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha, agent_id, team_name, workflow_name, parent_session_id, agent_type, description, spawn_depth"

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
}

// scanner abstracts *sql.Row and *sql.Rows for sessionFacetRow.scan.
type scanner interface {
	Scan(dest ...interface{}) error
}

func (sf *sessionFacetRow) scan(s scanner) error {
	return s.Scan(&sf.sessionID, &sf.email, &sf.branch, &sf.actorType, &sf.capturedAt,
		&sf.turnCount, &sf.toolCallCount, &sf.fileCount, &sf.checkpointID, &sf.gitSHA,
		&sf.agentID, &sf.teamName, &sf.workflowName, &sf.parentSessionID,
		&sf.agentType, &sf.description, &sf.spawnDepth)
}

// detail builds the JSON session detail; optional metadata fields are only
// set when present so they are omitted from output for agents without them.
func (sf *sessionFacetRow) detail(files []string) sessionDetail {
	return sessionDetail{
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
	}
}

func buildFilterWhere(filters RecallFilters) (string, []interface{}) {
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
		// File filter applied post-query via files_index.
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

// nomicSearch computes deep semantic similarity using nomic-embed-text embeddings.
// Non-fatal: returns nil on any failure or when nomic is unavailable.
func nomicSearch(indexDB *sql.DB, query string, gitRoot string) (map[string]float64, error) {
	if !nomic.Supported() {
		return nil, nil
	}

	// Load stored nomic embeddings.
	embeddings, err := db.QueryEmbeddings(indexDB, nomic.ModelName)
	if err != nil || len(embeddings) == 0 {
		return nil, err
	}

	// Load client and embed the query.
	client, err := nomic.NewClient(gitRoot)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	queryVec, err := client.EmbedQuery(query)
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

func buildResults(indexDB *sql.DB, scored []scored, filters RecallFilters, limit int) ([]searchResult, error) {
	// Compile file regex if present.
	var fileRe *regexp.Regexp
	if filters.File != "" {
		var err error
		fileRe, err = regexp.Compile(filters.File)
		if err != nil {
			return nil, fmt.Errorf("invalid file regex: %w", err)
		}
	}

	var results []searchResult
	for _, s := range scored {
		if len(results) >= limit {
			break
		}

		// Load session facets.
		var sf sessionFacetRow
		row := indexDB.QueryRow(
			"SELECT "+sessionFacetCols+" FROM session_facets WHERE session_id = $1",
			s.sessionID,
		)
		if err := sf.scan(row); err != nil {
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

		files, _ := querySessionFiles(indexDB, s.sessionID)

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

		results = append(results, searchResult{
			SessionID:      s.sessionID,
			Score:          math.Round(s.score*100) / 100,
			Snippet:        snippet,
			SnippetTurnIdx: snippetIdx,
			SnippetRole:    snippetRole,
			Session:        sf.detail(files),
		})
	}

	return results, nil
}

// loadParentIDs batch-loads parent_session_id for a set of candidate
// sessions. Sessions with no parent (or no harness concept of one) map to
// "" — treated identically to a top-level trunk session downstream.
func loadParentIDs(indexDB *sql.DB, sessionIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(sessionIDs))
	args := make([]interface{}, len(sessionIDs))
	for i, sid := range sessionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sid
		result[sid] = "" // default: no parent, until proven otherwise
	}

	rows, err := indexDB.Query(
		"SELECT session_id, parent_session_id FROM session_facets WHERE session_id IN ("+strings.Join(placeholders, ",")+")",
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
func groupByConversation(indexDB *sql.DB, results []searchResult, parentIDs map[string]string, limit int) []searchResult {
	rootCache := make(map[string]string, len(parentIDs))
	for sid, parent := range parentIDs {
		rootCache[sid] = parent
	}

	type group struct {
		root    string
		members []searchResult
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

	out := make([]searchResult, 0, limit)
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
		out = append(out, searchResult{
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
func loadSessionFacet(indexDB *sql.DB, sessionID string) (sessionDetail, bool) {
	var sf sessionFacetRow
	row := indexDB.QueryRow("SELECT "+sessionFacetCols+" FROM session_facets WHERE session_id = $1", sessionID)
	if err := sf.scan(row); err != nil {
		return sessionDetail{}, false
	}
	files, _ := querySessionFiles(indexDB, sessionID)
	return sf.detail(files), true
}

type scored struct {
	sessionID string
	score     float64
	hit       *sessionHit
}

type sessionHit struct {
	bestHit    bm25Hit
	bm25Max    float64
	lsaScore   float64
	nomicScore float64
}

func sortScored(s []scored) {
	// Sort descending by score.
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].score > s[j-1].score; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
		content = content[:defaultSnippetSize] + "..."
	}
	return content, turnIndex, role
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
		return content[:defaultSnippetSize] + "..."
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

	// Align to word boundaries.
	if start > 0 {
		for start < end && content[start] != ' ' {
			start++
		}
		start++ // skip the space
	}
	if end < len(content) {
		for end > start && content[end-1] != ' ' {
			end--
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
