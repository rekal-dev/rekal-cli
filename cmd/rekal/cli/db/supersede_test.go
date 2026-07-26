package db

import (
	"database/sql"
	"fmt"
	"testing"
)

// seedSession inserts a session and n turns whose contents come from body.
func seedSession(t *testing.T, d *sql.DB, id, source, email string, body []string) {
	t.Helper()
	if err := InsertSession(d, id, "", "h-"+id, "human", "", email, "main", "2026-07-11T00:00:00Z", source); err != nil {
		t.Fatalf("InsertSession %s: %v", id, err)
	}
	for i, c := range body {
		if err := InsertTurn(d, fmt.Sprintf("%s:%d", id, i), id, i, "human", c, ""); err != nil {
			t.Fatalf("InsertTurn %s:%d: %v", id, i, err)
		}
	}
}

// TestPurgeSuperseded_CollapsesRecaptures covers the duplicate-session bug: before
// capture keyed on transcript identity, every commit during a live session stored
// the whole conversation again. The ledger keeps those copies (append-only, and
// already on the wire), so the index must collapse a chain of prefixes down to the
// longest — while never merging two conversations that merely start alike.
func TestPurgeSuperseded_CollapsesRecaptures(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// One conversation captured at three commits — a prefix chain.
	grow := []string{"start the work", "keep going", "and finish"}
	seedSession(t, dataDB, "cap1", "claude", "a@b.c", grow[:1])
	seedSession(t, dataDB, "cap2", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "cap3", "claude", "a@b.c", grow)

	// Same opening turn, different agent — must survive.
	seedSession(t, dataDB, "other", "codex", "a@b.c", grow[:1])
	// Same opening turn, same agent, but it diverges — must survive.
	seedSession(t, dataDB, "fork", "claude", "a@b.c", []string{"start the work", "a different path"})
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	live := map[string]bool{}
	rows, err := indexDB.Query(`SELECT DISTINCT session_id FROM turns_ft`)
	if err != nil {
		t.Fatalf("query turns_ft: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		live[id] = true
	}
	rows.Close() //nolint:errcheck

	for _, gone := range []string{"cap1", "cap2"} {
		if live[gone] {
			t.Errorf("%s is a strict prefix of cap3 and should have been collapsed", gone)
		}
	}
	for _, kept := range []string{"cap3", "other", "fork"} {
		if !live[kept] {
			t.Errorf("%s must survive: it is either the longest capture or a distinct conversation", kept)
		}
	}
}

// TestSupersededSessionIDs_DataDB covers the export-side use of the same
// detection. It reads the data DB's own tables rather than the index's, and a
// silent failure there is worse than a loud one: export would ship the
// duplicates the filter exists to withhold, onto a branch other people fetch.
func TestSupersededSessionIDs_DataDB(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close() //nolint:errcheck
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	grow := []string{"opening turn", "second", "third"}
	seedSession(t, dataDB, "short", "claude", "a@b.c", grow[:1])
	seedSession(t, dataDB, "mid", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "full", "claude", "a@b.c", grow)
	seedSession(t, dataDB, "elsewhere", "codex", "a@b.c", grow[:1])

	got, err := SupersededSessionIDs(dataDB)
	if err != nil {
		t.Fatalf("SupersededSessionIDs: %v", err)
	}
	for _, id := range []string{"short", "mid"} {
		if !got[id] {
			t.Errorf("%s is a strict prefix of full and must be withheld from the wire", id)
		}
	}
	for _, id := range []string{"full", "elsewhere"} {
		if got[id] {
			t.Errorf("%s must be shipped: it is the longest capture or a different agent's session", id)
		}
	}
}
