package db

import (
	"database/sql"
	"fmt"
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

// InsertKnowledgeChunks bulk-inserts chunk rows.
func InsertKnowledgeChunks(d *sql.DB, chunks []KnowledgeChunkRow) error {
	for _, c := range chunks {
		if _, err := d.Exec(`
			INSERT INTO knowledge_chunks
				(id, path, anchor, breadcrumb, start_line, end_line, content, content_hash, blob_sha)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			c.ID, c.Path, c.Anchor, c.Breadcrumb, c.StartLine, c.EndLine,
			c.Content, c.ContentHash, c.BlobSHA,
		); err != nil {
			return fmt.Errorf("insert knowledge chunk %s: %w", c.ID, err)
		}
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
