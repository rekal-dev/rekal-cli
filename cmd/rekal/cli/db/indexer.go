package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// LoadFTSExtension loads the DuckDB FTS extension.
// On the four shipped platforms (linux/amd64, linux/arm64, darwin/amd64,
// darwin/arm64) the extension is embedded in the binary and extracted to a
// local cache — no network access required. The linux/amd64 and darwin/arm64
// blobs are committed; the linux/arm64 and darwin/amd64 blobs are downloaded
// at build time by scripts/fetch-fts-extension.sh. On any other platform
// ftsExtensionGZ is empty and it falls back to downloading from the HTTPS
// repository at first use.
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
		// Write to a unique temp file and rename into place: two rekal
		// processes can race the first extraction (post-commit hook and a
		// recall, or parallel tests), and a direct WriteFile leaves the
		// loser LOADing a half-written extension. Rename is atomic and both
		// writers carry identical bytes, so last-writer-wins is safe.
		tmp, err := os.CreateTemp(cacheDir, "fts-*.duckdb_extension.tmp")
		if err != nil {
			return fmt.Errorf("create fts temp file: %w", err)
		}
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()           //nolint:errcheck
			os.Remove(tmp.Name()) //nolint:errcheck
			return fmt.Errorf("write fts extension: %w", err)
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmp.Name()) //nolint:errcheck
			return fmt.Errorf("close fts temp file: %w", err)
		}
		if err := os.Rename(tmp.Name(), extPath); err != nil {
			os.Remove(tmp.Name()) //nolint:errcheck
			return fmt.Errorf("install fts extension: %w", err)
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

// migrateDataAt opens the data DB read-write just long enough to apply
// forward-only schema migrations, so index population can rely on the
// current column set even for data DBs written by older rekal versions.
func migrateDataAt(gitRoot string) error {
	dataDB, err := OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data db for migration: %w", err)
	}
	defer dataDB.Close() //nolint:errcheck
	if err := MigrateDataSchema(dataDB); err != nil {
		return fmt.Errorf("migrate data db: %w", err)
	}
	return nil
}

// SummaryFingerprint is the stable prefix Claude Code puts on every
// compaction-summary turn. New checkpoints store these with role "summary"
// natively (the parser reads the isCompactSummary flag), but the flag never
// reaches data.db rows written by older rekal versions — or rows encoded by
// an old binary, which never tags the role byte. data.db is append-only and
// never rewritten, so the reclassification lives on the derived side instead:
// index population and the data-side drill-down rewrite the role on read
// (transport's sync import applies the same rule), and one `rekal index`
// after an upgrade re-tags all history.
//
// The reclassification is scoped to source = 'claude' sessions: the
// boilerplate is Claude Code's, and a Codex/Gemini/OpenCode turn that merely
// starts with the same text (e.g. pasted) must keep its stored role. Wire
// imports carry no source and are stored as 'claude' (InsertSessionMeta's
// default), so teammate frames remain covered.
const SummaryFingerprint = "This session is being continued from a previous conversation"

// summaryRoleExprJoined applies the reclassification in queries that join
// data_db.turns t with data_db.sessions s (index population).
const summaryRoleExprJoined = `CASE WHEN t.role = 'human' AND s.source = 'claude' AND t.content LIKE '` + SummaryFingerprint + `%' THEN 'summary' ELSE t.role END`

// summaryRoleExprData applies it in single-table queries against the data
// DB's turns, resolving the session's source with a correlated subquery
// (drill-down reads; one session at a time, so the subquery is cheap).
const summaryRoleExprData = `CASE WHEN role = 'human' AND content LIKE '` + SummaryFingerprint + `%' AND (SELECT source FROM sessions WHERE id = session_id) = 'claude' THEN 'summary' ELSE role END`

// PopulateIndex attaches the data DB and bulk-populates all index tables.
func PopulateIndex(d *sql.DB, gitRoot string) error {
	dataPath := filepath.Join(StoreDir(gitRoot), "data.db")

	// The data DB may have been written by an older rekal (it is shared via
	// git); make sure schema migrations ran before querying new columns.
	if err := migrateDataAt(gitRoot); err != nil {
		return err
	}

	if _, err := d.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		return fmt.Errorf("attach data_db: %w", err)
	}
	defer d.Exec("DETACH data_db") //nolint:errcheck

	// turns_ft — LEFT JOIN so a turn with a missing session row (anomalous)
	// still indexes, with its stored role.
	if _, err := d.Exec(`
		INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
		SELECT t.id, t.session_id, t.turn_index, ` + summaryRoleExprJoined + `, t.content, CAST(t.ts AS VARCHAR)
		FROM data_db.turns t
		LEFT JOIN data_db.sessions s ON s.id = t.session_id
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
			checkpoint_id, git_sha, parent_session_id, team_name, workflow_name,
			agent_type, description, spawn_depth
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
			c.git_sha,
			s.parent_session_id,
			s.team_name,
			s.workflow_name,
			s.agent_type,
			s.description,
			s.spawn_depth
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

	// Facet documents for the optional facet ranking layer — derived from
	// the index tables populated above.
	if err := PopulateFacetText(d); err != nil {
		return err
	}

	// BUG 6 residual: drop RekalBench harness sessions that already landed in
	// data.db before SkipCapture. Index-only — data.db stays append-only.
	if err := PurgeBenchSessionsFromIndex(d); err != nil {
		return err
	}

	return nil
}

// PurgeBenchSessionsFromIndex removes RekalBench harness sessions from the
// derived index (BUG 6 residual). Capture now skips them via
// session.SkipCapture; this cleans corpora that already imported fixtures.
// data.db is untouched (append-only).
func PurgeBenchSessionsFromIndex(d *sql.DB) error {
	ids := map[string]struct{}{}
	for _, fp := range session.BenchFingerprints() {
		rows, err := d.Query(`
			SELECT DISTINCT session_id FROM turns_ft
			WHERE role IN ('human', 'human_steering') AND content LIKE '%' || $1 || '%'`, fp)
		if err != nil {
			return fmt.Errorf("find bench sessions: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close() //nolint:errcheck
				return fmt.Errorf("scan bench session: %w", err)
			}
			ids[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close() //nolint:errcheck
			return err
		}
		rows.Close() //nolint:errcheck
	}
	if len(ids) == 0 {
		return nil
	}
	list := make([]string, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	return deleteSessionsFromIndex(d, list)
}

func deleteSessionsFromIndex(d *sql.DB, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	parts := make([]string, len(sessionIDs))
	args := make([]interface{}, len(sessionIDs))
	for i, id := range sessionIDs {
		parts[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	in := strings.Join(parts, ",")
	tables := []string{
		"turns_ft",
		"tool_calls_index",
		"files_index",
		"session_facets",
		"session_embeddings",
	}
	for _, table := range tables {
		q := fmt.Sprintf("DELETE FROM %s WHERE session_id IN (%s)", table, in)
		if _, err := d.Exec(q, args...); err != nil {
			// session_embeddings may be absent on very old indexes — ignore.
			if table == "session_embeddings" && strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return fmt.Errorf("purge %s: %w", table, err)
		}
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

// PopulateFacetText builds each session's facet document — distinct tool
// paths + command prefixes + steering-turn text, concatenated and capped —
// into session_facets.facet_text. It reads the index DB's own tables
// (tool_calls_index, turns_ft), so it covers native, synced, and cross-repo
// imported sessions uniformly; call it after those tables are populated.
// With no sessionIDs it rebuilds every row; with sessionIDs it updates only
// those (the incremental path). Deterministic, no LLM — the facet layer's
// evidence is tool-call metadata, not generated text.
func PopulateFacetText(d *sql.DB, sessionIDs ...string) error {
	// Cap keeps facet docs bounded: enough for thousands of distinct paths'
	// worth of signal, small enough that the FTS index stays cheap.
	const facetTextCap = 4000

	query := fmt.Sprintf(`
		UPDATE session_facets SET facet_text = substr(concat_ws(' ',
			(SELECT string_agg(DISTINCT tc.path, ' ')
			 FROM tool_calls_index tc
			 WHERE tc.session_id = session_facets.session_id
			   AND tc.path IS NOT NULL AND tc.path <> ''),
			(SELECT string_agg(DISTINCT tc.cmd_prefix, ' ')
			 FROM tool_calls_index tc
			 WHERE tc.session_id = session_facets.session_id
			   AND tc.cmd_prefix IS NOT NULL AND tc.cmd_prefix <> ''),
			(SELECT string_agg(t.content, ' ' ORDER BY t.turn_index)
			 FROM turns_ft t
			 WHERE t.session_id = session_facets.session_id
			   AND t.role = 'human_steering')
		), 1, %d)`, facetTextCap)

	var args []interface{}
	if len(sessionIDs) > 0 {
		inClause, inArgs := sqlPlaceholders(sessionIDs)
		query += " WHERE session_id IN (" + inClause + ")"
		args = inArgs
	}
	if _, err := d.Exec(query, args...); err != nil {
		return fmt.Errorf("populate facet_text: %w", err)
	}
	return nil
}

// sqlPlaceholders builds a "$1,$2,...,$N" list and matching args for an
// IN (...) clause, assuming $1 is the statement's first bound parameter.
func sqlPlaceholders(vals []string) (string, []interface{}) {
	placeholders := make([]string, len(vals))
	args := make([]interface{}, len(vals))
	for i, v := range vals {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = v
	}
	return strings.Join(placeholders, ","), args
}

// CreateFacetFTSIndex creates the DuckDB FTS index over the per-session
// facet documents (session_facets.facet_text). Guarded: when no session has
// facet material the index is not built at all, and the facet layer's query
// in search fails soft — so corpora without tool metadata pay nothing.
func CreateFacetFTSIndex(d *sql.DB) error {
	var count int
	err := d.QueryRow(`SELECT count(*) FROM session_facets
		WHERE facet_text IS NOT NULL AND facet_text <> ''`).Scan(&count)
	if err != nil || count == 0 {
		return nil //nolint:nilerr // guarded build: no facet docs, no index
	}
	if _, err := d.Exec(`PRAGMA create_fts_index('session_facets', 'session_id', 'facet_text', stemmer='english', stopwords='english', overwrite=1)`); err != nil {
		return fmt.Errorf("create facet fts index: %w", err)
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

// ReadIndexState returns the value for key, or ("", false, nil) when absent.
func ReadIndexState(d *sql.DB, key string) (string, bool, error) {
	var value string
	err := d.QueryRow("SELECT value FROM index_state WHERE key = $1", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read index_state: %w", err)
	}
	return value, true, nil
}

// ListSemanticModels returns distinct deep-semantic model names stored in
// session_embeddings (excludes the LSA model key). Ordered for stable logs.
func ListSemanticModels(d *sql.DB) ([]string, error) {
	rows, err := d.Query(`
		SELECT DISTINCT model FROM session_embeddings
		WHERE model != 'lsa-v1'
		ORDER BY model
	`)
	if err != nil {
		return nil, fmt.Errorf("list semantic models: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var models []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan semantic model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
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

// QueryEmbeddings returns session_id → embedding vector for a given model.
func QueryEmbeddings(d *sql.DB, model string) (map[string][]float64, error) {
	rows, err := d.Query("SELECT session_id, embedding FROM session_embeddings WHERE model = $1", model)
	if err != nil {
		return nil, fmt.Errorf("query embeddings: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string][]float64)
	for rows.Next() {
		var sid string
		var raw interface{}
		if err := rows.Scan(&sid, &raw); err != nil {
			return nil, fmt.Errorf("scan embedding: %w", err)
		}
		emb, err := toFloat64Slice(raw)
		if err != nil {
			return nil, fmt.Errorf("convert embedding for %s: %w", sid, err)
		}
		result[sid] = emb
	}
	return result, rows.Err()
}

// toFloat64Slice converts a DuckDB FLOAT[] result (returned as []interface{})
// into a []float64.
func toFloat64Slice(v interface{}) ([]float64, error) {
	switch arr := v.(type) {
	case []float64:
		return arr, nil
	case []interface{}:
		out := make([]float64, len(arr))
		for i, elem := range arr {
			switch n := elem.(type) {
			case float64:
				out[i] = n
			case float32:
				out[i] = float64(n)
			default:
				return nil, fmt.Errorf("unexpected element type %T at index %d", elem, i)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unexpected embedding type %T", v)
	}
}

// PopulateIndexIncremental adds new sessions to the index without a full rebuild.
// sessionIDs are the newly captured sessions. checkpointID is the new checkpoint.
func PopulateIndexIncremental(d *sql.DB, gitRoot string, sessionIDs []string, checkpointID string) error {
	dataPath := filepath.Join(StoreDir(gitRoot), "data.db")

	// See PopulateIndex: migrate before querying new columns.
	if err := migrateDataAt(gitRoot); err != nil {
		return err
	}

	if _, err := d.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		return fmt.Errorf("attach data_db: %w", err)
	}
	defer d.Exec("DETACH data_db") //nolint:errcheck

	for _, sid := range sessionIDs {
		// turns_ft
		if _, err := d.Exec(`
			INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			SELECT t.id, t.session_id, t.turn_index, `+summaryRoleExprJoined+`, t.content, CAST(t.ts AS VARCHAR)
			FROM data_db.turns t
			LEFT JOIN data_db.sessions s ON s.id = t.session_id
			WHERE t.session_id = $1
		`, sid); err != nil {
			return fmt.Errorf("incremental turns_ft: %w", err)
		}

		// tool_calls_index
		if _, err := d.Exec(`
			INSERT INTO tool_calls_index (id, session_id, call_order, tool, path, cmd_prefix)
			SELECT id, session_id, call_order, tool, path, cmd_prefix
			FROM data_db.tool_calls WHERE session_id = $1
		`, sid); err != nil {
			return fmt.Errorf("incremental tool_calls_index: %w", err)
		}

		// session_facets
		if _, err := d.Exec(`
			INSERT INTO session_facets (
				session_id, user_email, git_branch, actor_type, agent_id,
				captured_at, turn_count, tool_call_count, file_count,
				checkpoint_id, git_sha, parent_session_id, team_name, workflow_name,
				agent_type, description, spawn_depth
			)
			SELECT
				s.id, s.user_email,
				COALESCE(c.git_branch, s.branch),
				s.actor_type, s.agent_id, s.captured_at,
				(SELECT count(*) FROM data_db.turns t WHERE t.session_id = s.id),
				(SELECT count(*) FROM data_db.tool_calls tc WHERE tc.session_id = s.id),
				COALESCE(fc.cnt, 0),
				c.id, c.git_sha,
				s.parent_session_id, s.team_name, s.workflow_name,
				s.agent_type, s.description, s.spawn_depth
			FROM data_db.sessions s
			LEFT JOIN data_db.checkpoint_sessions cs ON cs.session_id = s.id
			LEFT JOIN data_db.checkpoints c ON c.id = cs.checkpoint_id
			LEFT JOIN (
				SELECT cs2.session_id, count(DISTINCT ft.file_path) AS cnt
				FROM data_db.checkpoint_sessions cs2
				JOIN data_db.files_touched ft ON ft.checkpoint_id = cs2.checkpoint_id
				WHERE cs2.session_id = $1
				GROUP BY cs2.session_id
			) fc ON fc.session_id = s.id
			WHERE s.id = $1
		`, sid); err != nil {
			return fmt.Errorf("incremental session_facets: %w", err)
		}
	}

	// files_index for the new checkpoint
	if _, err := d.Exec(`
		INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type)
		SELECT ft.checkpoint_id, cs.session_id, ft.file_path, ft.change_type
		FROM data_db.files_touched ft
		JOIN data_db.checkpoint_sessions cs ON cs.checkpoint_id = ft.checkpoint_id
		WHERE ft.checkpoint_id = $1
	`, checkpointID); err != nil {
		return fmt.Errorf("incremental files_index: %w", err)
	}

	// Facet documents for the new sessions. Like turns_ft, new rows become
	// searchable by the facet layer via the FTS index maintained at full
	// rebuild time (CreateFacetFTSIndex in index/sync).
	if len(sessionIDs) > 0 {
		if err := PopulateFacetText(d, sessionIDs...); err != nil {
			return err
		}
	}

	return nil
}

// QuerySessionContentByIDs returns session_id → concatenated turn content for specific sessions.
func QuerySessionContentByIDs(d *sql.DB, sessionIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(sessionIDs))
	for _, sid := range sessionIDs {
		// string_agg over zero rows returns NULL — a session can have tool
		// calls but no turns, and that must not fail the whole batch.
		var content sql.NullString
		err := d.QueryRow(`
			SELECT string_agg(content, ' ' ORDER BY turn_index)
			FROM turns_ft WHERE session_id = $1
		`, sid).Scan(&content)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("query session content for %s: %w", sid, err)
		}
		if content.Valid && content.String != "" {
			result[sid] = content.String
		}
	}
	return result, nil
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
