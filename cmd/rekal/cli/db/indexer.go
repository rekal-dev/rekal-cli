package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadFTSExtension loads the DuckDB FTS extension.
// On supported platforms (darwin/arm64, linux/amd64) the extension is embedded
// in the binary and extracted to a local cache — no network access required.
// On other platforms it falls back to downloading from the HTTPS repository.
func LoadFTSExtension(d *sql.DB) error {
	if len(ftsExtensionGZ) > 0 {
		return loadEmbeddedFTS(d)
	}
	return loadRemoteFTS(d)
}

func loadEmbeddedFTS(d *sql.DB) error {
	cacheDir, err := ftsCacheDir()
	if err != nil {
		return fmt.Errorf("fts cache dir: %w", err)
	}
	extPath := filepath.Join(cacheDir, "fts.duckdb_extension")

	// Only extract if not already cached.
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		gz, err := gzip.NewReader(bytes.NewReader(ftsExtensionGZ))
		if err != nil {
			return fmt.Errorf("decompress fts extension: %w", err)
		}
		defer gz.Close() //nolint:errcheck

		data, err := io.ReadAll(gz)
		if err != nil {
			return fmt.Errorf("read fts extension: %w", err)
		}

		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return fmt.Errorf("create fts cache dir: %w", err)
		}
		if err := os.WriteFile(extPath, data, 0o644); err != nil {
			return fmt.Errorf("write fts extension: %w", err)
		}
	}

	// LOAD from explicit path — no INSTALL or network needed.
	escaped := strings.ReplaceAll(extPath, "'", "''")
	if _, err := d.Exec(fmt.Sprintf("LOAD '%s'", escaped)); err != nil {
		return fmt.Errorf("load fts: %w", err)
	}
	return nil
}

func loadRemoteFTS(d *sql.DB) error {
	if _, err := d.Exec("SET custom_extension_repository='https://extensions.duckdb.org'"); err != nil {
		return fmt.Errorf("set extension repository: %w", err)
	}
	if _, err := d.Exec("INSTALL fts"); err != nil {
		return fmt.Errorf("install fts: %w", err)
	}
	if _, err := d.Exec("LOAD fts"); err != nil {
		return fmt.Errorf("load fts: %w", err)
	}
	return nil
}

// ftsCacheDir returns a directory for caching the extracted FTS extension.
func ftsCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "rekal", "extensions"), nil
}

// DropIndexTables drops all index tables for a clean rebuild.
func DropIndexTables(d *sql.DB) error {
	tables := []string{
		"index_state",
		"session_embeddings",
		"file_cooccurrence",
		"session_facets",
		"files_index",
		"tool_calls_index",
		"turns_ft",
	}
	for _, t := range tables {
		if _, err := d.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", t)); err != nil {
			return fmt.Errorf("drop table %s: %w", t, err)
		}
	}
	return nil
}

// PopulateIndex attaches the data DB and bulk-populates all index tables.
func PopulateIndex(d *sql.DB, gitRoot string) error {
	dataPath := filepath.Join(gitRoot, ".rekal", "data.db")

	if _, err := d.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		return fmt.Errorf("attach data_db: %w", err)
	}
	defer d.Exec("DETACH data_db") //nolint:errcheck

	// turns_ft
	if _, err := d.Exec(`
		INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
		SELECT id, session_id, turn_index, role, content, CAST(ts AS VARCHAR)
		FROM data_db.turns
	`); err != nil {
		return fmt.Errorf("populate turns_ft: %w", err)
	}

	// tool_calls_index
	if _, err := d.Exec(`
		INSERT INTO tool_calls_index (id, session_id, call_order, tool, path, cmd_prefix)
		SELECT id, session_id, call_order, tool, path, cmd_prefix
		FROM data_db.tool_calls
	`); err != nil {
		return fmt.Errorf("populate tool_calls_index: %w", err)
	}

	// files_index — denormalize session_id via checkpoint_sessions
	if _, err := d.Exec(`
		INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type)
		SELECT ft.checkpoint_id, cs.session_id, ft.file_path, ft.change_type
		FROM data_db.files_touched ft
		JOIN data_db.checkpoint_sessions cs ON cs.checkpoint_id = ft.checkpoint_id
	`); err != nil {
		return fmt.Errorf("populate files_index: %w", err)
	}

	// files_index — supplement from file-modifying tool_calls for existing data
	// that was checkpointed before the capture-time fix.
	gitRootPrefix := gitRoot + "/"
	if _, err := d.Exec(`
		INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type)
		SELECT DISTINCT cs.checkpoint_id, tc.session_id,
			replace(tc.path, $1, ''),
			'T'
		FROM data_db.tool_calls tc
		JOIN data_db.checkpoint_sessions cs ON cs.session_id = tc.session_id
		WHERE tc.tool IN ('Write', 'Edit', 'NotebookEdit')
		  AND tc.path IS NOT NULL AND length(tc.path) > 0
		  AND tc.path LIKE ($1 || '%')
		  AND NOT EXISTS (
			SELECT 1 FROM files_index fi
			WHERE fi.checkpoint_id = cs.checkpoint_id
			  AND fi.session_id = tc.session_id
			  AND fi.file_path = replace(tc.path, $1, '')
		  )
	`, gitRootPrefix); err != nil {
		return fmt.Errorf("populate files_index from tool_calls: %w", err)
	}

	// session_facets — aggregation
	if _, err := d.Exec(`
		INSERT INTO session_facets (
			session_id, user_email, git_branch, actor_type, agent_id,
			captured_at, turn_count, tool_call_count, file_count,
			checkpoint_id, git_sha
		)
		SELECT
			s.id,
			s.user_email,
			COALESCE(c.git_branch, s.branch),
			s.actor_type,
			s.agent_id,
			s.captured_at,
			(SELECT count(*) FROM data_db.turns t WHERE t.session_id = s.id),
			(SELECT count(*) FROM data_db.tool_calls tc WHERE tc.session_id = s.id),
			COALESCE(fc.file_count, 0),
			c.id,
			c.git_sha
		FROM data_db.sessions s
		LEFT JOIN data_db.checkpoint_sessions cs ON cs.session_id = s.id
		LEFT JOIN data_db.checkpoints c ON c.id = cs.checkpoint_id
		LEFT JOIN (
			SELECT cs2.session_id, count(DISTINCT ft.file_path) AS file_count
			FROM data_db.checkpoint_sessions cs2
			JOIN data_db.files_touched ft ON ft.checkpoint_id = cs2.checkpoint_id
			GROUP BY cs2.session_id
		) fc ON fc.session_id = s.id
	`); err != nil {
		return fmt.Errorf("populate session_facets: %w", err)
	}

	// file_cooccurrence — self-join on tool_calls paths within same session
	if _, err := d.Exec(`
		INSERT INTO file_cooccurrence (file_a, file_b, count)
		SELECT a.path, b.path, count(*) AS cnt
		FROM data_db.tool_calls a
		JOIN data_db.tool_calls b ON a.session_id = b.session_id AND a.path < b.path
		WHERE a.path IS NOT NULL AND a.path != ''
		  AND b.path IS NOT NULL AND b.path != ''
		GROUP BY a.path, b.path
	`); err != nil {
		return fmt.Errorf("populate file_cooccurrence: %w", err)
	}

	return nil
}

// CreateFTSIndex creates the DuckDB full-text search index on turns_ft.
func CreateFTSIndex(d *sql.DB) error {
	_, err := d.Exec(`PRAGMA create_fts_index('turns_ft', 'id', 'content', stemmer='english', stopwords='english', overwrite=1)`)
	if err != nil {
		return fmt.Errorf("create fts index: %w", err)
	}
	return nil
}

// IsIndexPopulated checks whether the index has been built.
func IsIndexPopulated(d *sql.DB) bool {
	var count int
	err := d.QueryRow("SELECT count(*) FROM index_state WHERE key = 'last_indexed_at'").Scan(&count)
	return err == nil && count > 0
}

// WriteIndexState writes a key-value pair to the index_state table.
func WriteIndexState(d *sql.DB, key, value string) error {
	_, err := d.Exec(`
		INSERT INTO index_state (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = $2
	`, key, value)
	if err != nil {
		return fmt.Errorf("write index_state: %w", err)
	}
	return nil
}

// StoreEmbeddings bulk-inserts session embeddings into the index DB.
func StoreEmbeddings(d *sql.DB, vectors map[string][]float64, model string) error {
	for sessionID, vec := range vectors {
		// Inline the array literal because the database/sql driver cannot
		// bind a string to a FLOAT[] column, even with a cast.
		query := fmt.Sprintf(
			`INSERT INTO session_embeddings (session_id, embedding, model, generated_at)
			 VALUES ($1, %s::FLOAT[], $2, now())`,
			float64SliceToDuckDB(vec),
		)
		if _, err := d.Exec(query, sessionID, model); err != nil {
			return fmt.Errorf("insert embedding for %s: %w", sessionID, err)
		}
	}
	return nil
}

// float64SliceToDuckDB serializes a float64 slice as a DuckDB list literal
// (e.g. "[0.1, 0.2, 0.3]") because the database/sql driver does not support
// passing Go slices for FLOAT[] columns.
func float64SliceToDuckDB(v []float64) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}

// QuerySessionContent returns session_id → concatenated turn content for LSA.
func QuerySessionContent(d *sql.DB) (map[string]string, error) {
	rows, err := d.Query(`
		SELECT session_id, string_agg(content, ' ' ORDER BY turn_index)
		FROM turns_ft
		GROUP BY session_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query session content: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string]string)
	for rows.Next() {
		var id, content string
		if err := rows.Scan(&id, &content); err != nil {
			return nil, fmt.Errorf("scan session content: %w", err)
		}
		result[id] = content
	}
	return result, rows.Err()
}
