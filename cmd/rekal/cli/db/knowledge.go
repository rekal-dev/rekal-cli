package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/scrub"
)

// knowledgeDDL defines the knowledge layer's index-side table: heading-anchored
// chunks of the repo's tracked prose files at HEAD
// (docs/design/knowledge-layer.md). Derived and local-only, like every index
// table — rebuilt at will, never pushed. Kept out of indexDDL so a refresh
// against an index.db built by an older rekal can create it on demand
// (EnsureKnowledgeSchema) without a full schema migration.
const knowledgeDDL = `
CREATE TABLE IF NOT EXISTS knowledge_chunks (
	id          VARCHAR PRIMARY KEY,
	path        VARCHAR NOT NULL,
	anchor      VARCHAR,
	breadcrumb  VARCHAR,
	start_line  INTEGER NOT NULL,
	end_line    INTEGER NOT NULL,
	content     VARCHAR NOT NULL,
	content_hash VARCHAR NOT NULL,
	blob_sha    VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kc_path ON knowledge_chunks(path);
CREATE INDEX IF NOT EXISTS idx_kc_hash ON knowledge_chunks(content_hash);

CREATE TABLE IF NOT EXISTS knowledge_embeddings (
	content_hash VARCHAR NOT NULL,
	model        VARCHAR NOT NULL,
	embedding    FLOAT[],
	PRIMARY KEY (content_hash, model)
);
`

// KnowledgeHeadKey is the index_state watermark: the commit SHA the knowledge
// layer was last built at. A recall whose HEAD matches it skips the refresh
// entirely (one rev-parse); a mismatch triggers an incremental re-chunk of
// only the files whose blobs changed.
const KnowledgeHeadKey = "knowledge_head_sha"

// KnowledgeChunkRow is one insert-ready chunk of a prose file.
type KnowledgeChunkRow struct {
	ID          string
	Path        string
	Anchor      string
	Breadcrumb  string
	StartLine   int
	EndLine     int
	Content     string
	ContentHash string
	BlobSHA     string
}

// EnsureKnowledgeSchema creates the knowledge table when missing — safe on
// every open, including index DBs built by older rekal versions.
func EnsureKnowledgeSchema(d *sql.DB) error {
	if _, err := d.Exec(knowledgeDDL); err != nil {
		return fmt.Errorf("create knowledge schema: %w", err)
	}
	return nil
}

// QueryKnowledgeBlobs returns path → blob SHA for every indexed prose file —
// the stored side of the change detector (compared against `git ls-tree -r
// HEAD` to find files needing a re-chunk).
func QueryKnowledgeBlobs(d *sql.DB) (map[string]string, error) {
	rows, err := d.Query(`SELECT DISTINCT path, blob_sha FROM knowledge_chunks`)
	if err != nil {
		return nil, fmt.Errorf("query knowledge blobs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string]string)
	for rows.Next() {
		var path, sha string
		if err := rows.Scan(&path, &sha); err != nil {
			return nil, fmt.Errorf("scan knowledge blob: %w", err)
		}
		result[path] = sha
	}
	return result, rows.Err()
}

// DeleteKnowledgeChunks removes every chunk of the given paths (changed or
// deleted files, ahead of a re-insert).
func DeleteKnowledgeChunks(d *sql.DB, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	inClause, args := sqlPlaceholders(paths)
	if _, err := d.Exec(`DELETE FROM knowledge_chunks WHERE path IN (`+inClause+`)`, args...); err != nil {
		return fmt.Errorf("delete knowledge chunks: %w", err)
	}
	return nil
}

// InsertKnowledgeChunks bulk-inserts chunk rows in one transaction — a full
// build on a docs-heavy repo inserts tens of thousands of chunks, and
// per-statement autocommit would dominate the first-index/first-hook cost.
func InsertKnowledgeChunks(d *sql.DB, chunks []KnowledgeChunkRow) error {
	if len(chunks) == 0 {
		return nil
	}
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("begin knowledge insert: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO knowledge_chunks
			(id, path, anchor, breadcrumb, start_line, end_line, content, content_hash, blob_sha)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`)
	if err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("prepare knowledge insert: %w", err)
	}
	defer stmt.Close() //nolint:errcheck
	for _, c := range chunks {
		// Last-line UTF-8/NUL guard (same as session scrub): prose files can
		// hold arbitrary bytes — a single bad .txt used to abort the whole
		// knowledge-layer transaction with "could not bind parameter".
		id := scrub.SanitizeText(c.ID)
		path := scrub.SanitizeText(c.Path)
		anchor := scrub.SanitizeText(c.Anchor)
		breadcrumb := scrub.SanitizeText(c.Breadcrumb)
		content := scrub.SanitizeText(c.Content)
		hash := c.ContentHash
		if content != c.Content || hash == "" {
			// Content changed under sanitize (or hash missing) — identity
			// must track the stored text for the embed-cache key.
			sum := sha256.Sum256([]byte(content))
			hash = hex.EncodeToString(sum[:])
		}
		blobSHA := scrub.SanitizeText(c.BlobSHA)
		if _, err := stmt.Exec(
			id, path, anchor, breadcrumb, c.StartLine, c.EndLine,
			content, hash, blobSHA,
		); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("insert knowledge chunk %s: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge insert: %w", err)
	}
	return nil
}

// QueryKnowledgeChunksMissingEmbeddings returns content_hash → content for
// chunks that have no stored vector under model, up to limit (0 = no limit).
// Distinct by hash — identical sections (or a moved file's chunks) embed
// once. This join is what lets embedding coverage converge across budgeted
// passes: each pass takes a bite, the next pass sees only what's left.
func QueryKnowledgeChunksMissingEmbeddings(d *sql.DB, model string, limit int) (map[string]string, error) {
	q := `
		SELECT DISTINCT kc.content_hash, kc.content
		FROM knowledge_chunks kc
		LEFT JOIN knowledge_embeddings ke
		  ON ke.content_hash = kc.content_hash AND ke.model = $1
		WHERE ke.content_hash IS NULL`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.Query(q, model)
	if err != nil {
		return nil, fmt.Errorf("query missing knowledge embeddings: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string]string)
	for rows.Next() {
		var hash, content string
		if err := rows.Scan(&hash, &content); err != nil {
			return nil, fmt.Errorf("scan missing knowledge embedding: %w", err)
		}
		result[hash] = content
	}
	return result, rows.Err()
}

// StoreKnowledgeEmbeddings inserts content_hash-keyed chunk vectors for a
// model. Conflicts (already stored) are ignored — hashes are content
// identities, so a re-store can never change a vector's meaning.
func StoreKnowledgeEmbeddings(d *sql.DB, vectors map[string][]float64, model string) error {
	for hash, vec := range vectors {
		query := fmt.Sprintf(
			`INSERT INTO knowledge_embeddings (content_hash, model, embedding)
			 VALUES ($1, $2, %s::FLOAT[])
			 ON CONFLICT (content_hash, model) DO NOTHING`,
			float64SliceToDuckDB(vec),
		)
		if _, err := d.Exec(query, hash, model); err != nil {
			return fmt.Errorf("insert knowledge embedding %s: %w", hash, err)
		}
	}
	return nil
}

// QueryKnowledgeEmbeddings returns content_hash → vector for a model.
func QueryKnowledgeEmbeddings(d *sql.DB, model string) (map[string][]float64, error) {
	rows, err := d.Query(`SELECT content_hash, embedding FROM knowledge_embeddings WHERE model = $1`, model)
	if err != nil {
		return nil, fmt.Errorf("query knowledge embeddings: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string][]float64)
	for rows.Next() {
		var hash string
		var raw interface{}
		if err := rows.Scan(&hash, &raw); err != nil {
			return nil, fmt.Errorf("scan knowledge embedding: %w", err)
		}
		vec, err := toFloat64Slice(raw)
		if err != nil {
			return nil, fmt.Errorf("convert knowledge embedding %s: %w", hash, err)
		}
		result[hash] = vec
	}
	return result, rows.Err()
}

// PruneKnowledgeEmbeddings drops vectors whose content no longer exists in
// any chunk (deleted or rewritten sections) — the embedcache still holds
// them, so a reverted edit re-fills from cache, never from the model.
func PruneKnowledgeEmbeddings(d *sql.DB) error {
	if _, err := d.Exec(`
		DELETE FROM knowledge_embeddings
		WHERE content_hash NOT IN (SELECT content_hash FROM knowledge_chunks)`); err != nil {
		return fmt.Errorf("prune knowledge embeddings: %w", err)
	}
	return nil
}

// CreateKnowledgeFTSIndex builds the FTS index over chunk content. Guarded
// like the facet index: a corpus with no prose files gets no index, and the
// knowledge search in the search package fails soft to an empty layer.
func CreateKnowledgeFTSIndex(d *sql.DB) error {
	var count int
	err := d.QueryRow(`SELECT count(*) FROM knowledge_chunks`).Scan(&count)
	if err != nil || count == 0 {
		return nil //nolint:nilerr // guarded build: no chunks, no index
	}
	if _, err := d.Exec(`PRAGMA create_fts_index('knowledge_chunks', 'id', 'content', stemmer='english', stopwords='english', overwrite=1)`); err != nil {
		return fmt.Errorf("create knowledge fts index: %w", err)
	}
	return nil
}
