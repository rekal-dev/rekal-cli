package search

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
)

// knowledge.go is the query side of the knowledge layer
// (docs/design/knowledge-layer.md): BM25 over heading-anchored chunks of the
// repo's tracked prose files, aggregated to file-level hits — chunks are
// scored, files are returned, mirroring how turns are scored and sessions
// returned. Hits are pointers (path + anchor + line range + snippet), never
// file content: the index finds, the agent's Read tool serves live from HEAD.
//
// The layer is additive and fails soft: no knowledge FTS index (a corpus
// with no prose files, or an index.db predating the layer) means an absent
// knowledge block, and the sessions ranking is untouched either way.

const (
	// knowledgeLimit caps returned file hits — thin output, agent drills.
	knowledgeLimit = 5
	// knowledgeAlsoLimit caps runner-up anchors per file.
	knowledgeAlsoLimit = 2
	// knowledgeSessionsLimit caps the provenance edge per file — same order
	// as relatedLimit: enough to zoom along, thin on the output.
	knowledgeSessionsLimit = 3
	// knowledgeCoverageBonus is the per-extra-chunk multiplier: a file
	// matching in several sections is more likely *the* document than
	// several files matching once.
	knowledgeCoverageBonus = 0.1
)

// KnowledgeAnchor is a runner-up section pointer within a knowledge hit.
type KnowledgeAnchor struct {
	Anchor string `json:"anchor,omitempty"`
	Lines  string `json:"lines"`
}

// KnowledgeHit is one file-level knowledge result: what the repo currently
// knows at HEAD, with the provenance edge (Sessions) into why it knows it.
type KnowledgeHit struct {
	Path    string `json:"path"`
	Anchor  string `json:"anchor,omitempty"`
	Lines   string `json:"lines"`
	Snippet string `json:"snippet"`
	// Also holds additional matching sections of the same file.
	Also []KnowledgeAnchor `json:"also,omitempty"`
	// LastModified is the abbreviated SHA of the last commit touching the
	// file — the freshness pointer.
	LastModified string `json:"last_modified,omitempty"`
	// Sessions are the ledger sessions that touched this file (files_index
	// join) — the "why we believe this" edge, drillable with session tools.
	Sessions []string `json:"sessions,omitempty"`
	Score    float64  `json:"score"`
}

// knowledgeChunkHit is one matching chunk before file-level aggregation.
type knowledgeChunkHit struct {
	path      string
	anchor    string
	startLine int
	endLine   int
	content   string
	score     float64
}

// knowledgeSearch runs the knowledge layer for a recall query. Never fails
// recall: any error (most commonly a missing knowledge FTS index) returns an
// absent layer.
func knowledgeSearch(indexDB *sql.DB, query, gitRoot string) []KnowledgeHit {
	rows, err := indexDB.Query(`
		SELECT kc.path, kc.anchor, kc.start_line, kc.end_line, kc.content,
		       fts_main_knowledge_chunks.match_bm25(kc.id, $1) AS score
		FROM knowledge_chunks kc
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT 100
	`, query)
	if err != nil {
		return nil // knowledge FTS index absent — layer off
	}
	defer rows.Close() //nolint:errcheck

	var hits []knowledgeChunkHit
	for rows.Next() {
		var h knowledgeChunkHit
		var anchor sql.NullString
		if err := rows.Scan(&h.path, &anchor, &h.startLine, &h.endLine, &h.content, &h.score); err != nil {
			return nil
		}
		h.anchor = nullStr(anchor)
		hits = append(hits, h)
	}
	if rows.Err() != nil || len(hits) == 0 {
		return nil
	}

	return aggregateKnowledge(indexDB, hits, query, gitRoot)
}

// aggregateKnowledge folds chunk hits (sorted by score descending) into
// file-level hits: best chunk drives the score and supplies snippet+anchor,
// runners-up become Also pointers, extra matching sections earn a coverage
// bonus.
func aggregateKnowledge(indexDB *sql.DB, hits []knowledgeChunkHit, query, gitRoot string) []KnowledgeHit {
	maxScore := hits[0].score
	for _, h := range hits {
		if h.score > maxScore {
			maxScore = h.score
		}
	}
	if maxScore <= 0 {
		return nil
	}

	type fileAgg struct {
		best   knowledgeChunkHit
		others []knowledgeChunkHit
	}
	order := make([]string, 0, len(hits))
	files := make(map[string]*fileAgg, len(hits))
	for _, h := range hits {
		f, ok := files[h.path]
		if !ok {
			files[h.path] = &fileAgg{best: h}
			order = append(order, h.path)
			continue
		}
		f.others = append(f.others, h)
	}

	out := make([]KnowledgeHit, 0, knowledgeLimit)
	scoreOf := func(f *fileAgg) float64 {
		norm := f.best.score / maxScore
		bonus := 1 + knowledgeCoverageBonus*math.Min(float64(len(f.others)), 2)
		return math.Min(norm*bonus, 1.0)
	}
	// Order paths by aggregated score (input order is best-chunk order, but
	// the coverage bonus can reorder ties and near-ties).
	sortByScore := func(paths []string) {
		for i := 1; i < len(paths); i++ {
			for j := i; j > 0 && scoreOf(files[paths[j]]) > scoreOf(files[paths[j-1]]); j-- {
				paths[j], paths[j-1] = paths[j-1], paths[j]
			}
		}
	}
	sortByScore(order)

	for _, path := range order {
		if len(out) >= knowledgeLimit {
			break
		}
		f := files[path]
		hit := KnowledgeHit{
			Path:         path,
			Anchor:       f.best.anchor,
			Lines:        fmt.Sprintf("%d-%d", f.best.startLine, f.best.endLine),
			Snippet:      extractSnippet(f.best.content, query),
			LastModified: gitx.LastCommitShort(gitRoot, path),
			Sessions:     knowledgeSessions(indexDB, path),
			Score:        round2(scoreOf(f)),
		}
		for i, o := range f.others {
			if i >= knowledgeAlsoLimit {
				break
			}
			hit.Also = append(hit.Also, KnowledgeAnchor{
				Anchor: o.anchor,
				Lines:  fmt.Sprintf("%d-%d", o.startLine, o.endLine),
			})
		}
		out = append(out, hit)
	}
	return out
}

// knowledgeSessions loads the provenance edge: sessions that touched the
// file, newest first (ULIDs are time-ordered, so id DESC is recency).
// Best-effort — an empty edge is a valid answer.
func knowledgeSessions(indexDB *sql.DB, path string) []string {
	rows, err := indexDB.Query(`
		SELECT DISTINCT session_id FROM files_index
		WHERE file_path = $1
		ORDER BY session_id DESC
		LIMIT `+fmt.Sprintf("%d", knowledgeSessionsLimit), path)
	if err != nil {
		return nil
	}
	defer rows.Close() //nolint:errcheck

	var sessions []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return nil
		}
		sessions = append(sessions, sid)
	}
	if rows.Err() != nil {
		return nil
	}
	return sessions
}
