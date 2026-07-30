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
	"sort"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// LoadFTSExtension loads the DuckDB FTS extension.
// On the three shipped platforms (linux/amd64, linux/arm64, darwin/arm64) the
// extension is embedded in the binary and extracted to a local cache — no
// network access required. The linux/amd64 and darwin/arm64 blobs are
// committed; the linux/arm64 blob is downloaded at build time by
// scripts/fetch-fts-extension.sh. On any other platform ftsExtensionGZ is
// empty and it falls back to downloading from the HTTPS repository at first
// use.
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
		-- One session spans several checkpoints once capture appends to it, so
		-- joining checkpoint_sessions directly would emit a row per checkpoint
		-- and violate session_facets' one-row-per-session key. Take the latest.
		LEFT JOIN (
			SELECT cs.session_id, c.id, c.git_sha, c.git_branch, c.ts,
			       row_number() OVER (PARTITION BY cs.session_id ORDER BY c.ts DESC) AS rn
			FROM data_db.checkpoint_sessions cs
			JOIN data_db.checkpoints c ON c.id = cs.checkpoint_id
		) c ON c.session_id = s.id AND c.rn = 1
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
	if err := PopulateCommitMessages(d, gitRoot); err != nil {
		return err
	}
	if err := PopulateFacetText(d); err != nil {
		return err
	}

	// BUG 6 residual: drop RekalBench harness sessions that already landed in
	// data.db before SkipCapture. Index-only — data.db stays append-only.
	if err := PurgeBenchSessionsFromIndex(d); err != nil {
		return err
	}

	// Collapse re-captures of one conversation: before capture learned to
	// append, every commit during a session stored the whole transcript again.
	// Must run before facets/reach so neither is computed for a session that
	// is about to be dropped.
	if err := PurgeSupersededSessionsFromIndex(d); err != nil {
		return err
	}

	// session_reach — L1 recall citation-graph aggregate, derived from
	// data.db.recall_edges.
	if err := PopulateSessionReach(d); err != nil {
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

// PurgeSupersededSessionsFromIndex removes sessions whose turn sequence is a
// strict prefix of a longer session's — the same conversation captured again at
// a later commit, before capture learned to append (see checkpoint.go).
//
// Capture used to key dedup on the transcript's content hash, so a session that
// grew by even one turn between commits was stored again in full. The ledger
// keeps every copy (append-only, and already on the wire), so the collapse has
// to happen here, in the derived index, where it is legal and reversible.
//
// Only strict prefixes are purged: same source and author, fewer turns, and
// every stored turn identical to the longer session's turn at the same index.
// Two distinct conversations that merely open alike are never merged.
// PurgeSupersededSessionsWithData runs the supersession pass with data_db
// attached for the duration. PopulateIndex detaches data_db when it returns,
// so a later caller — sync, once the remote branches have been imported — has
// no session metadata to group by and the pass silently does nothing. Callers
// outside PopulateIndex should use this rather than the bare purge.
func PurgeSupersededSessionsWithData(d *sql.DB, gitRoot string) error {
	dataPath := filepath.Join(StoreDir(gitRoot), "data.db")
	if _, err := d.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		return fmt.Errorf("attach data_db: %w", err)
	}
	defer d.Exec("DETACH data_db") //nolint:errcheck
	return PurgeSupersededSessionsFromIndex(d)
}

func PurgeSupersededSessionsFromIndex(d *sql.DB) error {
	superseded, err := supersededSessions(d, "turns_ft", "data_db.sessions", "session_facets")
	if err != nil {
		return err
	}
	// Re-point subagents before dropping their trunk. Under the duplicate bug a
	// trunk was re-stored at every commit, so a subagent captured mid-conversation
	// links to whichever copy existed then. Search groups sessions by walking
	// parent_session_id to the root, so leaving the pointer at a collapsed copy
	// detaches that subagent's work from its conversation. The survivor is the
	// same conversation, so children follow it.
	ids := make([]string, 0, len(superseded))
	for old := range superseded {
		ids = append(ids, old)
	}
	sort.Strings(ids) // deterministic order

	// Record the mapping so downstream derivations can follow the conversation
	// rather than the collapsed id — reach counts most of all.
	if err := EnsureReachSchema(d); err != nil {
		return err
	}
	if _, err := d.Exec(`DELETE FROM session_supersedes`); err != nil {
		return fmt.Errorf("clear session_supersedes: %w", err)
	}
	for _, old := range ids {
		if _, err := d.Exec(
			`INSERT INTO session_supersedes (old_session_id, survivor_session_id) VALUES ($1, $2)`,
			old, superseded[old],
		); err != nil {
			return fmt.Errorf("record supersession: %w", err)
		}
	}
	for _, old := range ids {
		// Rewritten as delete-then-insert rather than UPDATE: DuckDB can raise a
		// spurious duplicate-key error when updating a row on an indexed table,
		// even though the key column itself is untouched.
		survivor := superseded[old]
		if _, err := d.Exec(`CREATE OR REPLACE TEMP TABLE repoint AS
			SELECT * REPLACE ($1 AS parent_session_id)
			FROM session_facets WHERE parent_session_id = $2`, survivor, old); err != nil {
			return fmt.Errorf("stage subagent repoint: %w", err)
		}
		if _, err := d.Exec(
			`DELETE FROM session_facets WHERE parent_session_id = $1`, old); err != nil {
			return fmt.Errorf("clear stale subagent parents: %w", err)
		}
		if _, err := d.Exec(`INSERT INTO session_facets SELECT * FROM repoint`); err != nil {
			return fmt.Errorf("repoint subagent parents: %w", err)
		}
	}
	if _, err := d.Exec(`DROP TABLE IF EXISTS repoint`); err != nil {
		return fmt.Errorf("drop repoint temp: %w", err)
	}
	return deleteSessionsFromIndex(d, ids)
}

// SupersededSessionIDs reports, for a data DB, the sessions that are re-captures
// of a longer one. Export uses it to keep duplicates off the wire: the ledger
// keeps every copy, but the orphan branch is derived data (see push --re-export)
// and there is no reason to ship the same conversation several times.
func SupersededSessionIDs(d *sql.DB) (map[string]bool, error) {
	m, err := supersededSessions(d, "turns", "sessions", "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(m))
	for id := range m {
		out[id] = true
	}
	return out, nil
}

// SupersededSessionSurvivors maps each superseded session to the longer session
// that replaces it. Export uses it to re-point a subagent whose trunk is not
// being shipped, so the teammate does not receive a dangling parent.
func SupersededSessionSurvivors(d *sql.DB) (map[string]string, error) {
	return supersededSessions(d, "turns", "sessions", "")
}

// supersededSessions finds sessions whose turn sequence is a strict prefix of a
// longer session's, reading turns from turnsTbl and session metadata from
// sessTbl (which differ between the derived index and the data DB).
//
// facetsTbl, when non-empty, supplies metadata for sessions that sessTbl does
// not know about. In the index that is every synced teammate session: the wire
// import writes turns_ft and session_facets directly and never touches data.db,
// so a LEFT JOIN to data_db.sessions misses them and source/author/parent all
// read as empty. The group key still carries turn 0, so those sessions would be
// grouped on opening turn alone — which is precisely what the comment on the
// query below refuses to do, since two people who start from the same plan
// prompt are not the same conversation. Reading the discriminators from
// session_facets keeps the key honest for sessions the data DB never saw.
// sess is one candidate in the supersession pass: its grouping key, the
// fingerprint of its whole ordered transcript, and how many turns it has.
type sess struct {
	id     string
	key    string
	fp     string
	nTurns int
}

func supersededSessions(d *sql.DB, turnsTbl, sessTbl, facetsTbl string) (map[string]string, error) {
	// Only sessions from the same agent, author and parent can be re-captures
	// of one conversation; turn 0 must match too, since any true prefix shares
	// it. Different agents legitimately produce sessions with identical opening
	// turns, so source is part of the key, not an afterthought.
	facetJoin, facetOrigin, facetEmail, facetParent := "", "", "", ""
	if facetsTbl != "" {
		facetJoin = fmt.Sprintf("LEFT JOIN %s f ON f.session_id = t.session_id", facetsTbl)
		// origin labels cross-repo local imports, which are also absent from
		// the data DB; without it a session imported from another repo could
		// group with a wire-imported one that opens the same way.
		facetOrigin = "COALESCE(MAX(f.origin), '') || '\x1f' ||"
		facetEmail = "COALESCE(MAX(f.user_email), '') ||"
		facetParent = "COALESCE(MAX(f.parent_session_id), '') ||"
	}
	// The fingerprint is the whole ordered transcript hashed in one pass, so
	// exact duplicates fall out of a GROUP BY instead of a pairwise comparison
	// per candidate pair. role is included because a turn reclassified to
	// 'summary' is not the same turn.
	rows, err := d.Query(fmt.Sprintf(`
		SELECT t.session_id,
		       COALESCE(MAX(s.source), '') || '\x1f' ||
		       %[4]s
		       COALESCE(MAX(s.user_email), %[5]s '') || '\x1f' ||
		       COALESCE(MAX(s.parent_session_id), %[6]s '') || '\x1f' ||
		       COALESCE(MAX(CASE WHEN t.turn_index = 0 THEN t.content END), ''),
		       md5(string_agg(
		           COALESCE(t.role, '') || '\x1f' || COALESCE(t.content, ''),
		           '\x1e' ORDER BY t.turn_index)),
		       count(*)
		FROM %[1]s t
		LEFT JOIN %[2]s s ON s.id = t.session_id
		%[3]s
		GROUP BY t.session_id`,
		turnsTbl, sessTbl, facetJoin, facetOrigin, facetEmail, facetParent))
	if err != nil {
		// In the index path the data DB is only attached during PopulateIndex.
		// Without it the discriminators are unavailable, and collapsing on turn
		// 0 alone would merge unrelated conversations — so do nothing.
		//
		// Only that one case is benign. Swallowing every error here also hides
		// a malformed query, and the symptom is indistinguishable from "no
		// duplicates found" — the pass silently stops collapsing anything and
		// nothing says so. Narrow the catch to the missing attachment.
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil, nil //nolint:nilerr
		}
		return nil, fmt.Errorf("group sessions for supersession: %w", err)
	}
	var all []sess
	for rows.Next() {
		var s sess
		if err := rows.Scan(&s.id, &s.key, &s.fp, &s.nTurns); err != nil {
			rows.Close() //nolint:errcheck
			return nil, fmt.Errorf("scan session: %w", err)
		}
		all = append(all, s)
	}
	if err := rows.Err(); err != nil {
		rows.Close() //nolint:errcheck
		return nil, err
	}
	rows.Close() //nolint:errcheck

	// Only sessions sharing turn 0 can stand in a prefix relation.
	groups := map[string][]sess{}
	for _, s := range all {
		if s.key == "" {
			continue // no turn 0 indexed — cannot be judged a prefix
		}
		groups[s.key] = append(groups[s.key], s)
	}

	superseded := map[string]string{}

	// Collapse exact duplicates first — in memory, from the fingerprints the
	// grouping query already returned — so the prefix pass below sees one
	// representative per distinct transcript and the candidate set is as small
	// as it can be before any turn content is loaded.
	candidates := make([]sess, 0, len(all))
	pending := make([][]sess, 0, len(groups))
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		// Exact duplicates first. The prefix pass below only ever compares a
		// session against a *strictly longer* one, so identical copies never
		// meet it and every one of them survives — the same conversation
		// reaching the index twice by two routes (a wire import landing beside
		// the local capture it came from, two peers who both carry it, a
		// cross-repo import of a repo that already pushed) costs a recall seed
		// per copy. Equal fingerprint means equal transcript, so the choice of
		// survivor is arbitrary and only has to be deterministic: the smallest
		// id, which for ULIDs is the copy that was captured first.
		g = collapseEqualRecaptures(g, superseded)
		if len(g) < 2 {
			continue
		}
		sort.Slice(g, func(i, j int) bool { return g[i].nTurns < g[j].nTurns })
		pending = append(pending, g)
		candidates = append(candidates, g...)
	}

	// One bulk read of per-turn hashes for every remaining candidate, then all
	// prefix comparison in memory.
	//
	// This was a FULL OUTER JOIN over the turns table per candidate *pair*.
	// Pairs grow quadratically inside a group, so a store with a few hundred
	// duplicate candidates issued thousands of joins — measured at seconds
	// each on a real store, with data.db held open throughout, blocking every
	// other rekal process. The comparison never needed the database: it is a
	// prefix test over two ordered lists.
	//
	// Hashes rather than content keep the transfer and the resident set bounded
	// — a turn is often kilobytes, its hash is not — and role is folded in
	// because a turn reclassified to 'summary' is not the same turn.
	hashes, err := loadTurnHashes(d, turnsTbl, candidates)
	if err != nil {
		return nil, err
	}
	for _, g := range pending {
		// Compare each session against every strictly longer one, shortest
		// first, so a chain (39 → 40 → 144 → 172) collapses to its longest.
		for i := 0; i < len(g); i++ {
			for j := len(g) - 1; j > i; j-- {
				if g[j].nTurns <= g[i].nTurns {
					continue
				}
				if turnsArePrefix(hashes[g[i].id], hashes[g[j].id], g[i].nTurns) {
					superseded[g[i].id] = g[j].id
					break
				}
			}
		}
	}
	// Flatten. The two passes can chain — an exact copy points at its twin,
	// which the prefix pass then points at a longer capture — and every
	// consumer treats the map as old → *final* survivor: the index deletes
	// every key, so a value that is itself a key would re-point a subagent or
	// a reach count at a session that is no longer there.
	return flattenSupersession(superseded), nil
}

// collapseEqualRecaptures maps every session that shares its transcript
// fingerprint with an earlier one onto that one, recording the result in
// superseded, and returns the group with those copies removed so the prefix
// pass sees one representative per distinct transcript.
func collapseEqualRecaptures(g []sess, superseded map[string]string) []sess {
	byFP := map[string]string{} // fingerprint → surviving id
	kept := g[:0:0]
	for _, s := range g {
		winner, seen := byFP[s.fp]
		if !seen {
			byFP[s.fp] = s.id
			kept = append(kept, s)
			continue
		}
		// Deterministic and acyclic: the smallest id always wins, so a group
		// of mutually-identical copies converges on one survivor regardless of
		// the order the rows arrived in.
		if s.id < winner {
			byFP[s.fp] = s.id
			superseded[winner] = s.id
			for i := range kept {
				if kept[i].id == winner {
					kept[i] = s
					break
				}
			}
			continue
		}
		superseded[s.id] = winner
	}
	return kept
}

// flattenSupersession resolves old → survivor chains so every key maps to a
// session that is not itself superseded. The cycle guard is a safety net, not
// an expected case: both passes only ever point at a strictly longer session
// or a strictly smaller id, so neither can close a loop.
func flattenSupersession(superseded map[string]string) map[string]string {
	for old := range superseded {
		seen := map[string]bool{old: true}
		final := superseded[old]
		for {
			next, ok := superseded[final]
			if !ok || seen[final] {
				break
			}
			seen[final] = true
			final = next
		}
		superseded[old] = final
	}
	return superseded
}

// loadTurnHashes reads turn_index → hash for every candidate session in one
// query, so the prefix pass needs no further database work.
func loadTurnHashes(d *sql.DB, turnsTbl string, candidates []sess) (map[string]map[int]string, error) {
	if len(candidates) == 0 {
		return map[string]map[int]string{}, nil
	}
	ids := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if !seen[c.id] {
			seen[c.id] = true
			ids = append(ids, c.id)
		}
	}
	in, args := sqlPlaceholders(ids)
	rows, err := d.Query(fmt.Sprintf(`
		SELECT session_id, turn_index,
		       md5(COALESCE(role, '') || '\x1f' || COALESCE(content, ''))
		FROM %s WHERE session_id IN (%s)`, turnsTbl, in), args...)
	if err != nil {
		return nil, fmt.Errorf("load turn hashes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make(map[string]map[int]string, len(ids))
	for rows.Next() {
		var sid, hash string
		var idx int
		if err := rows.Scan(&sid, &idx, &hash); err != nil {
			return nil, fmt.Errorf("scan turn hash: %w", err)
		}
		m := out[sid]
		if m == nil {
			m = map[int]string{}
			out[sid] = m
		}
		m[idx] = hash
	}
	return out, rows.Err()
}

// turnsArePrefix reports whether short's first n turns are identical to long's
// turns at the same indices. Keyed by turn_index rather than position so a gap
// in either sequence can never silently align two different turns.
func turnsArePrefix(short, long map[int]string, n int) bool {
	if len(short) == 0 || len(long) == 0 {
		return false
	}
	matched := 0
	for idx, h := range short {
		if idx >= n {
			continue
		}
		lh, ok := long[idx]
		if !ok || lh != h {
			return false
		}
		matched++
	}
	return matched == n
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
			   AND t.role = 'human_steering'),
			session_facets.commit_message
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
		// Upsert: a session's vector is replaced when its content changes (it
		// grew since the last capture), so this must not fail on the existing
		// (session_id, model) row.
		query := fmt.Sprintf(
			`INSERT INTO session_embeddings (session_id, embedding, model, generated_at)
			 VALUES ($1, %s::FLOAT[], $2, now())
			 ON CONFLICT (session_id, model) DO UPDATE
			 SET embedding = EXCLUDED.embedding, generated_at = EXCLUDED.generated_at`,
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
		// Clear this session's derived rows before re-inserting. A session that
		// grew since its last checkpoint is indexed again, and blind inserts
		// collide on the primary key — aborting the incremental update and
		// leaving the index behind until a full rebuild. Re-deriving one
		// session's slice is always safe: the index is derived data, and
		// data.db remains the source of truth.
		//
		// session_embeddings is dropped too, and that matters more than it
		// looks. The embed pass skips any session that already has a vector, so
		// a conversation embedded at fifty turns and grown to five hundred would
		// keep answering semantic recall from the first fifty — no error, no
		// warning, just quietly worse retrieval on exactly the long sessions
		// worth retrieving. Dropping it here forces a recompute over the current
		// content. This costs no more than before capture learned to append,
		// when every re-capture created a new session and embedded it in full.
		for _, tbl := range []string{"turns_ft", "tool_calls_index", "session_facets", "files_index", "session_embeddings"} {
			if _, err := d.Exec(fmt.Sprintf("DELETE FROM %s WHERE session_id = $1", tbl), sid); err != nil {
				return fmt.Errorf("clear %s for reindex: %w", tbl, err)
			}
		}

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
			-- See PopulateIndex: latest checkpoint only, one row per session.
			LEFT JOIN (
				SELECT cs.session_id, c.id, c.git_sha, c.git_branch, c.ts,
				       row_number() OVER (PARTITION BY cs.session_id ORDER BY c.ts DESC) AS rn
				FROM data_db.checkpoint_sessions cs
				JOIN data_db.checkpoints c ON c.id = cs.checkpoint_id
				WHERE cs.session_id = $1
			) c ON c.session_id = s.id AND c.rn = 1
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

	// Facet documents for the new sessions.
	if len(sessionIDs) > 0 {
		if err := PopulateCommitMessages(d, gitRoot); err != nil {
			return err
		}
		if err := PopulateFacetText(d, sessionIDs...); err != nil {
			return err
		}
	}

	// session_reach — recompute the L1 aggregate from data.db.recall_edges so
	// edges drained into data.db at this checkpoint become visible to recall.
	if err := PopulateSessionReach(d); err != nil {
		return err
	}

	// Refresh the BM25 snapshot. DuckDB builds the FTS index by PRAGMA over the
	// table's current contents and does not maintain it as rows are written, so
	// without this the turns just inserted sit in turns_ft while match_bm25 —
	// the only turn-level path recall has — scores nothing for them. Every
	// checkpoint's work stayed invisible to recall until someone happened to run
	// a full `rekal index`: in the ledger, unfindable, and nothing to signal it.
	// overwrite=1 makes this a rebuild rather than an error.
	//
	// Last, and only the extension load fails soft. This work is the one step
	// here that can be redone from the table alone, so a failure costs a stale
	// snapshot and nothing else; every step above writes rows keyed to *this*
	// checkpoint's sessions, and there is no watermark that would bring them
	// back. Returning early from the middle used to strand exactly those rows —
	// a conversation that never grew again kept its turns and lost its facets
	// until someone ran a full `rekal index`. The extension itself is a soft
	// skip because it is a precondition, not a result: without it there is no
	// snapshot to refresh, and the full-rebuild path loads it in index_cmd.
	if err := LoadFTSExtension(d); err == nil {
		if ftsErr := CreateFTSIndex(d); ftsErr != nil {
			return fmt.Errorf("refresh fts after incremental index: %w", ftsErr)
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

// PopulateCommitMessages fills session_facets.commit_message from this clone's
// git history, for every session anchored to a commit.
//
// The message is derived, never stored: SOUL.md's "strip what git already has"
// rules it out of data.db and off the wire, because every clone carries it
// already. That is also what makes it work for teammates — the wire ships only
// the SHA, and each reader resolves the text against their own git, for zero
// added bytes.
//
// Fails soft. A SHA this clone has not fetched simply yields no text, and the
// facet document is built from whatever else the session has.
func PopulateCommitMessages(d *sql.DB, gitRoot string) error {
	rows, err := d.Query(`SELECT DISTINCT git_sha FROM session_facets
		WHERE git_sha IS NOT NULL AND git_sha <> ''`)
	if err != nil {
		return fmt.Errorf("list anchored commits: %w", err)
	}
	var shas []string
	for rows.Next() {
		var sha string
		if err := rows.Scan(&sha); err != nil {
			rows.Close() //nolint:errcheck
			return fmt.Errorf("scan git_sha: %w", err)
		}
		shas = append(shas, sha)
	}
	if err := rows.Err(); err != nil {
		rows.Close() //nolint:errcheck
		return err
	}
	rows.Close() //nolint:errcheck
	if len(shas) == 0 {
		return nil
	}

	messages := gitx.CommitMessages(gitRoot, shas)
	if len(messages) == 0 {
		return nil
	}

	if _, err := d.Exec(`CREATE OR REPLACE TEMP TABLE commit_msgs (
		git_sha VARCHAR, message VARCHAR)`); err != nil {
		return fmt.Errorf("stage commit messages: %w", err)
	}
	for sha, msg := range messages {
		if _, err := d.Exec(`INSERT INTO commit_msgs VALUES ($1, $2)`, sha, msg); err != nil {
			return fmt.Errorf("stage commit message: %w", err)
		}
	}
	// One set-based UPDATE, the same shape PopulateFacetText uses. A per-row
	// UPDATE keyed on the primary key is what trips DuckDB's spurious
	// duplicate-key error (see the anchor write in transport/sync.go).
	if _, err := d.Exec(`UPDATE session_facets
		SET commit_message = (SELECT m.message FROM commit_msgs m
			WHERE m.git_sha = session_facets.git_sha)
		WHERE git_sha IS NOT NULL AND git_sha <> ''`); err != nil {
		return fmt.Errorf("apply commit messages: %w", err)
	}
	return nil
}
