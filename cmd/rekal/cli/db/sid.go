package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
)

// Short session handles (s1, s2, …) are derived at query time from
// index.db's session_facets, ordered by ULID. They shrink agent-facing
// digests and --session args; the ULID remains the stored identity and
// is never rewritten on the wire.

var shortSessionHandleRe = regexp.MustCompile(`^s([1-9][0-9]*)$`)

// IsShortSessionHandle reports whether s looks like a query-time short
// handle (s1, s2, …) rather than a ULID or other opaque id.
func IsShortSessionHandle(s string) bool {
	return shortSessionHandleRe.MatchString(s)
}

// SessionSIDMap is the bidirectional ULID ↔ sN map for one index.db.
// Built once per recall/query; never persisted.
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
