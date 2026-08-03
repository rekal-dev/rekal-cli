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
