package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// The embedding cache decouples embedding cost from index rebuilds. index.db
// is disposable and rebuilt from scratch by `rekal index`/`rekal sync`, which
// used to mean re-embedding the entire corpus every rebuild. The cache keeps
// one vector per (model, content_hash) in its own small DB, so a rebuild
// embeds only content it has never seen.
//
// Keying by model means switching embedding backends/models invalidates by
// construction — the new model simply has no entries — and costs exactly one
// full re-embed. The cache stores only vectors, never session text, so the
// disposable-index principle holds: the only durable copies of the data stay
// in data.db and the agent's own transcript files.
//
// The cache is an optimization, never a dependency: every caller treats a
// missing or unopenable cache as "embed everything".

// EmbedCachePath returns the cache DB path inside .rekal/.
func EmbedCachePath(gitRoot string) string {
	return filepath.Join(gitRoot, ".rekal", "embed-cache.db")
}

// OpenEmbedCache opens (creating if needed) the embedding cache DB and
// ensures its schema.
func OpenEmbedCache(gitRoot string) (*sql.DB, error) {
	d, err := open(EmbedCachePath(gitRoot))
	if err != nil {
		return nil, err
	}
	if _, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS embed_cache (
			model        VARCHAR NOT NULL,
			content_hash VARCHAR NOT NULL,
			embedding    FLOAT[],
			PRIMARY KEY (model, content_hash)
		)`); err != nil {
		d.Close()
		return nil, fmt.Errorf("create embed_cache schema: %w", err)
	}
	return d, nil
}

// CacheGetEmbeddings returns the cached vectors for the given content hashes
// under model. Absent hashes are simply missing from the result.
func CacheGetEmbeddings(d *sql.DB, model string, hashes []string) (map[string][]float64, error) {
	out := make(map[string][]float64, len(hashes))
	const chunk = 512
	for start := 0; start < len(hashes); start += chunk {
		end := min(start+chunk, len(hashes))
		batch := hashes[start:end]

		placeholders := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)+1)
		args = append(args, model)
		for i, h := range batch {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, h)
		}
		if err := scanCacheRows(d, out,
			`SELECT content_hash, embedding FROM embed_cache
			 WHERE model = $1 AND content_hash IN (`+strings.Join(placeholders, ",")+`)`, args...); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// scanCacheRows runs one chunked cache query and folds the rows into out.
func scanCacheRows(d *sql.DB, out map[string][]float64, query string, args ...interface{}) error {
	rows, err := d.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query embed cache: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var h string
		var raw interface{}
		if err := rows.Scan(&h, &raw); err != nil {
			return fmt.Errorf("scan embed cache: %w", err)
		}
		vec, err := toFloat64Slice(raw)
		if err != nil {
			return fmt.Errorf("convert cached embedding: %w", err)
		}
		out[h] = vec
	}
	return rows.Err()
}

// CachePutEmbeddings stores vectors under model, keyed by content hash.
// Existing entries are left as-is (same key ⇒ same content ⇒ same vector).
func CachePutEmbeddings(d *sql.DB, model string, vectors map[string][]float64) error {
	for hash, vec := range vectors {
		query := fmt.Sprintf(
			`INSERT INTO embed_cache (model, content_hash, embedding)
			 VALUES ($1, $2, %s::FLOAT[])
			 ON CONFLICT (model, content_hash) DO NOTHING`,
			float64SliceToDuckDB(vec),
		)
		if _, err := d.Exec(query, model, hash); err != nil {
			return fmt.Errorf("insert embed cache entry: %w", err)
		}
	}
	return nil
}
