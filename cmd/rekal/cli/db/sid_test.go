package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsShortSessionHandle(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"s1":                         true,
		"s12":                        true,
		"s0":                         false,
		"s01":                        false,
		"S1":                         false,
		"01ARZ3NDEKTSV4RRFFQ69G5FAV": false,
		"":                           false,
		"s":                          false,
		"sid":                        false,
	}
	for in, want := range cases {
		if got := IsShortSessionHandle(in); got != want {
			t.Errorf("IsShortSessionHandle(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLoadSessionSIDMap_StableByULIDOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := MigrateIndexSchema(indexDB); err != nil {
		t.Fatalf("MigrateIndexSchema: %v", err)
	}

	// Insert out of ULID order; numbering must follow ORDER BY session_id
	// (…FAV < …FAW < …FB0 → s1, s2, s3).
	ids := []string{
		"01ARZ3NDEKTSV4RRFFQ69G5FB0",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
	}
	for _, id := range ids {
		if _, err := indexDB.Exec(`
			INSERT INTO session_facets (
				session_id, actor_type, captured_at, turn_count, tool_call_count, file_count
			) VALUES ($1, 'human', '2026-07-18T00:00:00Z', 1, 0, 0)
		`, id); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}

	m, err := LoadSessionSIDMap(indexDB)
	if err != nil {
		t.Fatalf("LoadSessionSIDMap: %v", err)
	}
	want := map[string]string{
		"01ARZ3NDEKTSV4RRFFQ69G5FAV": "s1",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW": "s2",
		"01ARZ3NDEKTSV4RRFFQ69G5FB0": "s3",
	}
	for ulid, sid := range want {
		if got := m.SID(ulid); got != sid {
			t.Errorf("SID(%s) = %q, want %q", ulid, got, sid)
		}
		got, err := m.Resolve(sid)
		if err != nil {
			t.Errorf("Resolve(%s): %v", sid, err)
			continue
		}
		if got != ulid {
			t.Errorf("Resolve(%s) = %q, want %q", sid, got, ulid)
		}
	}

	if _, err := m.Resolve("s99"); err == nil {
		t.Fatal("Resolve(s99) should fail")
	}
	ulid, err := m.Resolve("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err != nil || ulid != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("Resolve(ULID) = %q, %v", ulid, err)
	}
}

func TestLoadPersistedSessionSIDMap_NoneSaved(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	m, err := LoadPersistedSessionSIDMap(dir)
	if err != nil {
		t.Fatalf("LoadPersistedSessionSIDMap: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil map when nothing was ever saved, got %+v", m)
	}
}

// TestSessionSIDMap_PersistRoundTrip pins the fix for a real race: sN is
// derived at query time from index.db's current session_facets, but a
// background embed or rebuild can atomically swap that file between a
// recall printing sN and an agent drilling it — recomputing fresh at drill
// time would then silently resolve the same sN to a different session. The
// save/load pair here must round-trip byte-for-byte so `query -s` can pin
// resolution to exactly what a digest showed, not whatever the live index
// looks like later.
func TestSessionSIDMap_PersistRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	shown := &SessionSIDMap{
		ToULID: map[string]string{
			"s1": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"s2": "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		},
	}
	if err := SaveSessionSIDMap(dir, shown); err != nil {
		t.Fatalf("SaveSessionSIDMap: %v", err)
	}

	loaded, err := LoadPersistedSessionSIDMap(dir)
	if err != nil {
		t.Fatalf("LoadPersistedSessionSIDMap: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected a persisted map, got nil")
	}
	for sid, ulid := range shown.ToULID {
		if got := loaded.ToULID[sid]; got != ulid {
			t.Errorf("ToULID[%s] = %q, want %q", sid, got, ulid)
		}
		if got := loaded.SID(ulid); got != sid {
			t.Errorf("SID(%s) = %q, want %q", ulid, got, sid)
		}
	}

	// A later live rebuild that reassigns sN must not silently change what
	// the pinned map resolves an already-shown handle to.
	resolved, err := loaded.Resolve("s1")
	if err != nil || resolved != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("Resolve(s1) = %q, %v, want the pinned ULID unchanged", resolved, err)
	}
}

func TestSaveSessionSIDMap_NilIsNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveSessionSIDMap(dir, nil); err != nil {
		t.Fatalf("SaveSessionSIDMap(nil): %v", err)
	}
	m, err := LoadPersistedSessionSIDMap(dir)
	if err != nil || m != nil {
		t.Fatalf("expected nothing persisted after a nil save, got %+v, %v", m, err)
	}
}
