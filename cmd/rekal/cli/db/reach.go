package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// reachDDL defines the index-side aggregate of the L1 recall citation graph:
// one row per session that has ever been reached, with the edge counts and a
// representative query. Derived and local-only — rebuilt from
// data.db.recall_edges. Kept out of indexDDL so a refresh against an index.db
// built by an older rekal can create it on demand (EnsureReachSchema) without a
// schema-version bump. See docs/design/recall-graph.md.
//
// The two counts are kept apart because they are different evidence.
// reach_count includes every edge, and a recall edge only says the engine
// ranked this session into some window — its own past output, which is why
// feeding it back into ranking is a loop. drill_count counts the edges where an
// agent chose to open the session: evidence from outside the ranker. Collapsing
// both into one number let the weak signal bury the strong one (measured on a
// real store: 741 recall edges against 6 drills, with 36 of 37 sessions
// "reached" and the top slot held by a three-turn session).
const reachDDL = `
CREATE TABLE IF NOT EXISTS session_reach (
	target_session_id VARCHAR PRIMARY KEY,
	reach_count       INTEGER NOT NULL DEFAULT 0,
	drill_count       INTEGER NOT NULL DEFAULT 0,
	last_query        VARCHAR,
	top_query         VARCHAR,
	last_ts           TIMESTAMP
);
`

// reachColumnUpgrades add the post-launch columns to a session_reach that an
// older rekal already created. The table is derived, but it is created on
// demand rather than by a versioned migration, so the old shape survives until
// something widens it. DuckDB rejects a constrained ADD COLUMN, so these carry
// no NOT NULL and every read coalesces.
var reachColumnUpgrades = []string{
	`ALTER TABLE session_reach ADD COLUMN IF NOT EXISTS drill_count INTEGER DEFAULT 0`,
	`ALTER TABLE session_reach ADD COLUMN IF NOT EXISTS top_query VARCHAR`,
}

// EnsureReachSchema creates session_reach when missing — safe on every open,
// including index DBs built by older rekal versions.
func EnsureReachSchema(d *sql.DB) error {
	if _, err := d.Exec(reachDDL); err != nil {
		return fmt.Errorf("create reach schema: %w", err)
	}
	for _, stmt := range reachColumnUpgrades {
		if _, err := d.Exec(stmt); err != nil {
			return fmt.Errorf("upgrade reach schema: %w", err)
		}
	}
	// session_supersedes maps a collapsed re-capture to the surviving copy of
	// the same conversation. Reach is keyed by whichever session id was
	// surfaced at recall time, which under the duplicate bug was whatever copy
	// existed then — so without this fold a heavily used conversation reads as
	// never used the moment its copies collapse.
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS session_supersedes (
		old_session_id      VARCHAR PRIMARY KEY,
		survivor_session_id VARCHAR NOT NULL
	)`); err != nil {
		return fmt.Errorf("create session_supersedes: %w", err)
	}
	return nil
}

// RecallEdge is one insert-ready row of the recall citation graph.
type RecallEdge struct {
	ID     string
	TS     time.Time
	Kind   string // "recall" | "drill"
	Query  string
	Target string // target session id
}

// InsertRecallEdges appends edges to data.db.recall_edges (append-only). Empty
// input is a no-op.
func InsertRecallEdges(d *sql.DB, edges []RecallEdge) error {
	if len(edges) == 0 {
		return nil
	}
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("begin recall_edges: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO recall_edges (id, ts, kind, query, target_session_id)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare recall_edges: %w", err)
	}
	defer stmt.Close() //nolint:errcheck
	for _, e := range edges {
		if _, err := stmt.Exec(e.ID, e.TS, e.Kind, e.Query, e.Target); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert recall_edge: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recall_edges: %w", err)
	}
	return nil
}

// Reach is the per-session aggregate the recall hint reads: how many times a
// session was surfaced, how many times an agent actually opened it, and the
// query that surfaced it most often.
type Reach struct {
	Count  int    // every edge — recall and drill
	Drills int    // edges where an agent opened the session
	Query  string // the query that reached it most often, ties broken by recency
	LastTS time.Time
}

// LoadReach returns the reach aggregate for the given target session ids,
// reading the derived index.db.session_reach table. Sessions with no history
// are simply absent from the map. Best-effort: a missing session_reach table
// (index built by an older rekal, never populated) yields an empty map, not an
// error, so recall degrades to no-hint rather than failing.
func LoadReach(d *sql.DB, ids []string) (map[string]Reach, error) {
	out := make(map[string]Reach)
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT target_session_id, reach_count, COALESCE(drill_count, 0),
			COALESCE(top_query, last_query), last_ts
		FROM session_reach
		WHERE target_session_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := d.Query(q, args...)
	if err != nil {
		// Table may not exist yet, or predate drill_count/top_query on an index
		// this binary has not rebuilt — treat as no data rather than failing the
		// recall that asked for a display hint.
		return out, nil //nolint:nilerr
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var (
			id     string
			count  int
			drills int
			query  sql.NullString
			ts     sql.NullTime
		)
		if err := rows.Scan(&id, &count, &drills, &query, &ts); err != nil {
			return nil, fmt.Errorf("scan reach: %w", err)
		}
		r := Reach{Count: count, Drills: drills, Query: query.String}
		if ts.Valid {
			r.LastTS = ts.Time
		}
		out[id] = r
	}
	return out, rows.Err()
}

// RefreshSessionReach opens the index at gitRoot, attaches data.db, and rebuilds
// session_reach. Used by the checkpoint path when recall-graph edges were
// drained but no new session was captured — so the incremental index update
// (which normally refreshes session_reach) does not run. No-op if the index DB
// doesn't exist yet. The data DB must not be held open by the caller (a second
// live handle to one DuckDB file in-process can crash under CGO).
func RefreshSessionReach(gitRoot string) error {
	if _, err := os.Stat(IndexPath(gitRoot)); err != nil {
		return nil // no index yet; next full build will populate it
	}
	indexDB, err := OpenIndex(gitRoot)
	if err != nil {
		return err
	}
	defer indexDB.Close() //nolint:errcheck
	dataPath := filepath.Join(StoreDir(gitRoot), "data.db")
	if _, err := indexDB.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		return fmt.Errorf("attach data_db: %w", err)
	}
	defer indexDB.Exec("DETACH data_db") //nolint:errcheck
	return PopulateSessionReach(indexDB)
}

// PopulateSessionReach rebuilds session_reach from data.db.recall_edges. It
// assumes the data DB is ATTACHed as data_db (as in PopulateIndex /
// PopulateIndexIncremental). The aggregate is one row per reached session, so a
// full recompute is cheap and always correct — used by both the full rebuild
// and the incremental checkpoint refresh.
//
// reach_count counts every edge and drill_count only the opened-it edges, so
// ranking can read the strong signal without losing the display count.
// top_query is the query that reached the session most often (ties broken by
// recency) rather than simply the newest: the hint is shown to an agent as how
// the need was framed, and last-writer-wins let one stray query relabel every
// row in the corpus. last_query/last_ts keep their literal meaning. Drills log
// no query, so both query columns come from recall edges.
func PopulateSessionReach(d *sql.DB) error {
	if err := EnsureReachSchema(d); err != nil {
		return err
	}
	if _, err := d.Exec(`DELETE FROM session_reach`); err != nil {
		return fmt.Errorf("clear session_reach: %w", err)
	}
	if _, err := d.Exec(`
		WITH edges AS (
			SELECT COALESCE(m.survivor_session_id, e.target_session_id) AS target_session_id,
			       e.kind, e.query, e.ts
			FROM data_db.recall_edges e
			LEFT JOIN session_supersedes m ON m.old_session_id = e.target_session_id
		),
		query_hits AS (
			SELECT target_session_id, query, count(*) AS hits, max(ts) AS newest
			FROM edges
			WHERE query IS NOT NULL AND query <> ''
			GROUP BY target_session_id, query
		),
		top_queries AS (
			SELECT target_session_id, arg_max(query, (hits, newest)) AS top_query
			FROM query_hits
			GROUP BY target_session_id
		)
		INSERT INTO session_reach (target_session_id, reach_count, drill_count, last_query, top_query, last_ts)
		SELECT
			e.target_session_id,
			count(*),
			count(*) FILTER (WHERE e.kind = 'drill'),
			arg_max(e.query, e.ts) FILTER (WHERE e.query IS NOT NULL AND e.query <> ''),
			any_value(t.top_query),
			max(e.ts)
		FROM edges e
		LEFT JOIN top_queries t ON t.target_session_id = e.target_session_id
		GROUP BY e.target_session_id
	`); err != nil {
		return fmt.Errorf("populate session_reach: %w", err)
	}
	return nil
}
