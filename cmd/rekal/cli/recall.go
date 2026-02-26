package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/cli/cmd/rekal/cli/lsa"
	"github.com/spf13/cobra"
)

const (
	defaultSnippetSize = 300
	defaultBM25Weight  = 0.4
	defaultLSAWeight   = 0.6
	defaultLimit       = 20
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
		results, err = hybridSearch(indexDB, filters, limit)
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

func hybridSearch(indexDB *sql.DB, filters RecallFilters, limit int) ([]searchResult, error) {
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

	// Step 3: Group by session, pick best turn per session.
	sessions := make(map[string]*sessionHit)

	for _, hit := range bm25Hits {
		sh, ok := sessions[hit.sessionID]
		if !ok {
			sh = &sessionHit{}
			sessions[hit.sessionID] = sh
		}
		if hit.score > sh.bm25Max {
			sh.bm25Max = hit.score
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

	// Compute hybrid scores.
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
		hybrid := defaultBM25Weight*bm25Norm + defaultLSAWeight*lsaNorm
		scoredResults = append(scoredResults, scored{sid, hybrid, sh})
	}

	// Sort by score descending.
	sortScored(scoredResults)

	// Apply filters and build results.
	return buildResults(indexDB, scoredResults, filters, limit)
}

func filterSearch(indexDB *sql.DB, filters RecallFilters, limit int) ([]searchResult, error) {
	// Build WHERE clause from filters.
	where, args := buildFilterWhere(filters)

	query := "SELECT session_id, user_email, git_branch, actor_type, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha FROM session_facets"
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
		if err := rows.Scan(&sf.sessionID, &sf.email, &sf.branch, &sf.actorType, &sf.capturedAt, &sf.turnCount, &sf.toolCallCount, &sf.fileCount, &sf.checkpointID, &sf.gitSHA); err != nil {
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
			Session: sessionDetail{
				Author:     nullStr(sf.email),
				Actor:      sf.actorType,
				Branch:     nullStr(sf.branch),
				CapturedAt: sf.capturedAt,
				Commit:     nullStr(sf.gitSHA),
				TurnCount:  sf.turnCount,
				ToolCalls:  sf.toolCallCount,
				Files:      files,
			},
		})
	}
	return results, rows.Err()
}

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
	// Load all embeddings.
	rows, err := indexDB.Query("SELECT session_id, embedding FROM session_embeddings")
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	embeddings := make(map[string][]float64)
	for rows.Next() {
		var sid string
		var emb []float64
		if err := rows.Scan(&sid, &emb); err != nil {
			return nil, err
		}
		embeddings[sid] = emb
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, nil
	}

	// We need the LSA model to project the query. Rebuild from session content.
	sessionContent, err := db.QuerySessionContent(indexDB)
	if err != nil {
		return nil, err
	}

	model, err := lsa.Build(sessionContent, lsa.DefaultDimension)
	if err != nil || model == nil {
		return nil, err
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
		err := indexDB.QueryRow(
			"SELECT session_id, user_email, git_branch, actor_type, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha FROM session_facets WHERE session_id = $1",
			s.sessionID,
		).Scan(&sf.sessionID, &sf.email, &sf.branch, &sf.actorType, &sf.capturedAt, &sf.turnCount, &sf.toolCallCount, &sf.fileCount, &sf.checkpointID, &sf.gitSHA)
		if err != nil {
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
			Session: sessionDetail{
				Author:     nullStr(sf.email),
				Actor:      sf.actorType,
				Branch:     nullStr(sf.branch),
				CapturedAt: sf.capturedAt,
				Commit:     nullStr(sf.gitSHA),
				TurnCount:  sf.turnCount,
				ToolCalls:  sf.toolCallCount,
				Files:      files,
			},
		})
	}

	return results, nil
}

type scored struct {
	sessionID string
	score     float64
	hit       *sessionHit
}

type sessionHit struct {
	bestHit  bm25Hit
	bm25Max  float64
	lsaScore float64
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
