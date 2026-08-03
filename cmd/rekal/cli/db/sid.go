package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// Short session handles (s1, s2, …) are derived at query time from
// index.db's session_facets, ordered by ULID. They shrink agent-facing
// digests and --session args; the ULID remains the stored identity and
// is never rewritten on the wire.
//
// The assignment is deterministic for a *fixed* index.db, but index.db is
// not fixed: recall auto-heals a stale/empty index inline, and a background
// `rekal embed` or `rekal sync` can atomically swap it out between the
// moment a digest is printed and the moment an agent drills the sN it
// named. Recomputing the map fresh at drill time — which is what this file
// used to do exclusively — silently resolves the same "sN" to a different
// session when that race lands, since ROW_NUMBER() shifts with the row set.
// SaveSessionSIDMap/LoadPersistedSessionSIDMap close that gap: whoever
// prints sN also pins the exact map that produced it, so a drill made
// against "what was just shown" reads that pinned map instead of
// recomputing live. See attachShortIDs (search.go) and its use in
// runSessionDrilldown (query.go).
var shortSessionHandleRe = regexp.MustCompile(`^s([1-9][0-9]*)$`)

// IsShortSessionHandle reports whether s looks like a query-time short
// handle (s1, s2, …) rather than a ULID or other opaque id.
func IsShortSessionHandle(s string) bool {
	return shortSessionHandleRe.MatchString(s)
}

// SessionSIDMap is the bidirectional ULID ↔ sN map for one index.db.
type SessionSIDMap struct {
	ToSID  map[string]string // ULID → sN
	ToULID map[string]string // sN → ULID
}

// LoadSessionSIDMap assigns s1..sN by ROW_NUMBER() OVER (ORDER BY
// session_id). ULIDs sort lexicographically ≈ mint time, so the mapping
// is deterministic for a given index contents.
func LoadSessionSIDMap(d *sql.DB) (*SessionSIDMap, error) {
	rows, err := d.Query(`
		SELECT session_id, ROW_NUMBER() OVER (ORDER BY session_id) AS n
		FROM session_facets
	`)
	if err != nil {
		return nil, fmt.Errorf("load session short ids: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	m := &SessionSIDMap{
		ToSID:  make(map[string]string),
		ToULID: make(map[string]string),
	}
	for rows.Next() {
		var ulid string
		var n int
		if err := rows.Scan(&ulid, &n); err != nil {
			return nil, fmt.Errorf("scan session short id: %w", err)
		}
		sid := "s" + strconv.Itoa(n)
		m.ToSID[ulid] = sid
		m.ToULID[sid] = ulid
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// Resolve returns the canonical ULID for handle. Short handles (sN) are
// looked up in the map; anything else is returned unchanged (assumed ULID).
func (m *SessionSIDMap) Resolve(handle string) (string, error) {
	if !IsShortSessionHandle(handle) {
		return handle, nil
	}
	if m == nil {
		return "", fmt.Errorf("short session handle %q: no sid map", handle)
	}
	ulid, ok := m.ToULID[handle]
	if !ok {
		return "", fmt.Errorf("unknown short session handle %q", handle)
	}
	return ulid, nil
}

// SID returns the short handle for ulid, or "" if unknown / map nil.
func (m *SessionSIDMap) SID(ulid string) string {
	if m == nil {
		return ""
	}
	return m.ToSID[ulid]
}

// shownSIDsPath is local-only (.rekal is gitignored wholesale), same as
// embed-cache.db and recall-log.ndjson — never on the wire.
func shownSIDsPath(gitRoot string) string {
	return filepath.Join(StoreDir(gitRoot), "shown_sids.json")
}

type shownSIDMapFile struct {
	ToULID map[string]string `json:"to_ulid"`
}

// SaveSessionSIDMap pins the sN→ULID map that just produced agent-visible
// output (a recall digest or its --json form) so a subsequent `query -s sN`
// resolves against exactly what was shown, not whatever index.db looks like
// by the time the drill runs.
//
// Written via a uniquely-named temp file in the same directory, then an
// atomic rename over the live path — the same pattern index_cmd.go uses for
// index.db and graph.go uses for the recall spool. A plain os.WriteFile
// truncates before it writes, so a reader (or a second recall's writer)
// landing mid-write would see a torn or empty file; two recalls close
// together are the common case here (auto-widening's own internal framing
// calls, or two agents against the same store), not a rare edge. The unique
// temp name means concurrent writers never clobber each other's in-progress
// file, and the rename means a reader always sees either the previous pin
// or the new one whole, never a partial one. Whichever rename lands last
// wins; there is no ordering requirement beyond that, since a drill that
// misses the pin already falls back to live resolution. Best-effort
// end-to-end: any failure here only costs the pin, not the recall itself.
func SaveSessionSIDMap(gitRoot string, m *SessionSIDMap) error {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(shownSIDMapFile{ToULID: m.ToULID})
	if err != nil {
		return err
	}
	dir := StoreDir(gitRoot)
	tmp, err := os.CreateTemp(dir, "shown_sids.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()        //nolint:errcheck // best-effort cleanup on the failure path
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup on the failure path
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup on the failure path
		return err
	}
	if err := os.Rename(tmpPath, shownSIDsPath(gitRoot)); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup on the failure path
		return err
	}
	return nil
}

// LoadPersistedSessionSIDMap reads the map saved by the most recent
// SaveSessionSIDMap in this repo. Returns nil, nil when there is nothing
// pinned yet (fresh store, or a store never written by a version that
// pins) — callers fall back to the live index-derived map.
func LoadPersistedSessionSIDMap(gitRoot string) (*SessionSIDMap, error) {
	data, err := os.ReadFile(shownSIDsPath(gitRoot))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f shownSIDMapFile
	if err := json.Unmarshal(data, &f); err != nil {
		// Corrupt cache: not worth failing a drill over. Caller falls back
		// to the live map.
		return nil, nil //nolint:nilerr
	}
	m := &SessionSIDMap{
		ToULID: f.ToULID,
		ToSID:  make(map[string]string, len(f.ToULID)),
	}
	for sid, ulid := range f.ToULID {
		m.ToSID[ulid] = sid
	}
	return m, nil
}
