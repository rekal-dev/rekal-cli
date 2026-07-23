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
// one row per session that has ever been reached (recalled or drilled), with a
// count and the most recent non-empty query that reached it. Derived and
// local-only — rebuilt from data.db.recall_edges. Kept out of indexDDL so a
// refresh against an index.db built by an older rekal can create it on demand
// (EnsureReachSchema) without a schema-version bump. See
// docs/design/recall-graph.md.
const reachDDL = `
CREATE TABLE IF NOT EXISTS session_reach (
	target_session_id VARCHAR PRIMARY KEY,
	reach_count       INTEGER NOT NULL DEFAULT 0,
	last_query        VARCHAR,
	last_ts           TIMESTAMP
);
`

// EnsureReachSchema creates session_reach when missing — safe on every open,
// including index DBs built by older rekal versions.
func EnsureReachSchema(d *sql.DB) error {
	if _, err := d.Exec(reachDDL); err != nil {
		return fmt.Errorf("create reach schema: %w", err)
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
// session was reached and a representative (most recent non-empty) query.
type Reach struct {
	Count  int
	Query  string
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
	q := `SELECT target_session_id, reach_count, last_query, last_ts
		FROM session_reach
		WHERE target_session_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := d.Query(q, args...)
	if err != nil {
		// Table may not exist yet on an un-populated index — treat as no data.
		return out, nil //nolint:nilerr
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var (
			id    string
			count int
			query sql.NullString
			ts    sql.NullTime
		)
		if err := rows.Scan(&id, &count, &query, &ts); err != nil {
			return nil, fmt.Errorf("scan reach: %w", err)
		}
		r := Reach{Count: count, Query: query.String}
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
// and the incremental checkpoint refresh. reach_count counts every edge;
// last_query is the newest non-empty query (drills log no query).
func PopulateSessionReach(d *sql.DB) error {
	if err := EnsureReachSchema(d); err != nil {
		return err
	}
	if _, err := d.Exec(`DELETE FROM session_reach`); err != nil {
		return fmt.Errorf("clear session_reach: %w", err)
	}
	if _, err := d.Exec(`
		INSERT INTO session_reach (target_session_id, reach_count, last_query, last_ts)
		SELECT
			target_session_id,
			count(*),
			arg_max(query, ts) FILTER (WHERE query IS NOT NULL AND query <> ''),
			max(ts)
		FROM data_db.recall_edges
		GROUP BY target_session_id
	`); err != nil {
		return fmt.Errorf("populate session_reach: %w", err)
	}
	return nil
}
