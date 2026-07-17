package search

import (
	"database/sql"
	"fmt"
	"math"
	"sort"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
)

// knowledge.go is the query side of the knowledge layer
// (docs/design/knowledge-layer.md): hybrid BM25 + deep-semantic scoring over
// heading-anchored chunks of the repo's tracked prose files, aggregated to
// file-level hits — chunks are scored, files are returned, mirroring how
// turns are scored and sessions returned. Hits are pointers (path + anchor +
// line range + snippet), never file content: the index finds, the agent's
// Read tool serves live from HEAD.
//
// The layer is additive and fails soft at every rung: no knowledge FTS index
// means an absent block; no chunk vectors (or no query vector) means
// keyword-only scoring — the two-speed contract. The sessions ranking is
// untouched either way.
//
// Latency: cosine runs only over BM25 candidate hashes (≤100), never the
// full embedding table. Semantic-only extras without keyword overlap are
// deferred until an ANN index exists — a full-corpus scan dominated recall
// (~half of wall time on prose-heavy repos).

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
	// knowledgeBM25Limit caps FTS candidates before semantic re-rank.
	knowledgeBM25Limit = 100
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

// knowledgeChunkHit is one candidate chunk before file-level aggregation.
type knowledgeChunkHit struct {
	path      string
	anchor    string
	startLine int
	endLine   int
	content   string
	hash      string
	bm25      float64
	semantic  float64
	score     float64 // combined, filled by knowledgeSearch
}

// knowledgeSearch runs the knowledge layer for a recall query. queryVec and
// model come from the session semantic pass when it ran (one query embed per
// recall, shared); when the session layer skipped but an HTTP query embedder
// exists, the query is embedded lazily — and only if chunk vectors for that
// model are actually stored. Never fails recall: any error returns an absent
// or keyword-only layer. The second return is scoring-lineage detail (nil
// when the layer is off) — never written to recall stdout JSON.
func knowledgeSearch(indexDB *sql.DB, query, gitRoot string, w Weights, qe QueryEmbedder, queryVec []float64, model string) ([]KnowledgeHit, []LineageKnowledgeHit) {
	hits := knowledgeBM25(indexDB, query)
	if len(hits) == 0 {
		return nil, nil
	}

	hashes := make([]string, len(hits))
	for i, h := range hits {
		hashes[i] = h.hash
	}
	semScores := knowledgeSemantic(indexDB, query, qe, queryVec, model, hashes)

	if len(semScores) == 0 {
		// Keyword-only: score is normalized BM25, the v1 behavior.
		var maxBM float64
		for _, h := range hits {
			if h.bm25 > maxBM {
				maxBM = h.bm25
			}
		}
		if maxBM <= 0 {
			return nil, nil
		}
		for i := range hits {
			hits[i].score = hits[i].bm25 / maxBM
		}
		return aggregateKnowledge(indexDB, hits, query, gitRoot)
	}

	// Hybrid: re-rank BM25 candidates with cosine. No full-corpus scan —
	// semantic-only (zero keyword overlap) chunks stay out until ANN.
	for i := range hits {
		hits[i].semantic = semScores[hits[i].hash]
	}

	var maxBM, maxSem float64
	for _, h := range hits {
		if h.bm25 > maxBM {
			maxBM = h.bm25
		}
		if h.semantic > maxSem {
			maxSem = h.semantic
		}
	}
	bmW, semW := w.layers2()
	for i := range hits {
		bmNorm, semNorm := 0.0, 0.0
		if maxBM > 0 {
			bmNorm = hits[i].bm25 / maxBM
		}
		if maxSem > 0 {
			semNorm = hits[i].semantic / maxSem
		}
		hits[i].score = bmW*bmNorm + semW*semNorm
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	return aggregateKnowledge(indexDB, hits, query, gitRoot)
}

// knowledgeBM25 returns chunk hits from the knowledge FTS index, best first.
// nil (layer off) when the index is absent — a corpus with no prose files,
// or an index.db predating the layer.
func knowledgeBM25(indexDB *sql.DB, query string) []knowledgeChunkHit {
	rows, err := indexDB.Query(`
		SELECT kc.path, kc.anchor, kc.start_line, kc.end_line, kc.content, kc.content_hash,
		       fts_main_knowledge_chunks.match_bm25(kc.id, $1) AS score
		FROM knowledge_chunks kc
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT `+fmt.Sprintf("%d", knowledgeBM25Limit), query)
	if err != nil {
		return nil
	}
	defer rows.Close() //nolint:errcheck

	var hits []knowledgeChunkHit
	for rows.Next() {
		var h knowledgeChunkHit
		var anchor sql.NullString
		if err := rows.Scan(&h.path, &anchor, &h.startLine, &h.endLine, &h.content, &h.hash, &h.bm25); err != nil {
			return nil
		}
		h.anchor = nullStr(anchor)
		hits = append(hits, h)
	}
	if rows.Err() != nil {
		return nil
	}
	return hits
}

// knowledgeSemantic returns content_hash → cosine similarity for the given
// candidate hashes only. Nil when the semantic rung is unavailable.
func knowledgeSemantic(indexDB *sql.DB, query string, qe QueryEmbedder, queryVec []float64, model string, hashes []string) map[string]float64 {
	if model == "" || len(hashes) == 0 {
		return nil
	}
	embeddings, err := db.QueryKnowledgeEmbeddingsByHashes(indexDB, model, hashes)
	if err != nil || len(embeddings) == 0 {
		return nil
	}
	if queryVec == nil {
		// The session pass didn't embed (e.g. a knowledge repo with no
		// session vectors yet). Embed lazily through the configured HTTP
		// backend; the embedded-nomic path stays keyword-only rather than
		// paying a second CGO model load.
		if qe == nil {
			return nil
		}
		if queryVec, err = qe.EmbedQuery(query); err != nil || queryVec == nil {
			return nil
		}
	}
	scores := make(map[string]float64, len(embeddings))
	for hash, emb := range embeddings {
		if sim := lsa.CosineSimilarity(queryVec, emb); sim > 0 {
			scores[hash] = sim
		}
	}
	return scores
}

// aggregateKnowledge folds chunk candidates (sorted by combined score
// descending) into file-level hits: best chunk drives the score and supplies
// snippet+anchor, runners-up become Also pointers, extra matching sections
// earn a coverage bonus. LineageKnowledgeHit carries the winning chunk's
// raw signals for scoring_lineage (never stdout).
func aggregateKnowledge(indexDB *sql.DB, hits []knowledgeChunkHit, query, gitRoot string) ([]KnowledgeHit, []LineageKnowledgeHit) {
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

	scoreOf := func(f *fileAgg) float64 {
		bonus := 1 + knowledgeCoverageBonus*math.Min(float64(len(f.others)), 2)
		return math.Min(f.best.score*bonus, 1.0)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return scoreOf(files[order[i]]) > scoreOf(files[order[j]])
	})

	out := make([]KnowledgeHit, 0, knowledgeLimit)
	lin := make([]LineageKnowledgeHit, 0, knowledgeLimit)
	for _, path := range order {
		if len(out) >= knowledgeLimit {
			break
		}
		f := files[path]
		score := round2(scoreOf(f))
		hit := KnowledgeHit{
			Path:         path,
			Anchor:       f.best.anchor,
			Lines:        fmt.Sprintf("%d-%d", f.best.startLine, f.best.endLine),
			Snippet:      extractSnippet(f.best.content, query),
			LastModified: gitx.LastCommitShort(gitRoot, path),
			Sessions:     knowledgeSessions(indexDB, path),
			Score:        score,
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
		lin = append(lin, LineageKnowledgeHit{
			Rank:     len(out),
			Path:     path,
			Anchor:   f.best.anchor,
			Score:    score,
			BM25:     round4(f.best.bm25),
			Semantic: round4(f.best.semantic),
		})
	}
	return out, lin
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
