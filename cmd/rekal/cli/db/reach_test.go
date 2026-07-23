package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecallEdgesReachAggregate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	t1 := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC) // newer
	edges := []RecallEdge{
		{ID: "e1", TS: t1, Kind: "recall", Query: "jwt expiry", Target: "S1"},
		{ID: "e2", TS: t2, Kind: "drill", Query: "", Target: "S1"}, // newer but empty query
		{ID: "e3", TS: t1, Kind: "recall", Query: "token refresh", Target: "S2"},
	}
	if err := InsertRecallEdges(dataDB, edges); err != nil {
		t.Fatalf("InsertRecallEdges: %v", err)
	}
	// Release the data.db writer before index attaches a second handle.
	if err := dataDB.Close(); err != nil {
		t.Fatalf("close data: %v", err)
	}

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}

	dataPath := filepath.Join(StoreDir(dir), "data.db")
	if _, err := indexDB.Exec(fmt.Sprintf("ATTACH '%s' AS data_db (READ_ONLY)", dataPath)); err != nil {
		t.Fatalf("attach data_db: %v", err)
	}
	if err := PopulateSessionReach(indexDB); err != nil {
		t.Fatalf("PopulateSessionReach: %v", err)
	}
	if _, err := indexDB.Exec("DETACH data_db"); err != nil {
		t.Fatalf("detach: %v", err)
	}

	reach, err := LoadReach(indexDB, []string{"S1", "S2", "S3"})
	if err != nil {
		t.Fatalf("LoadReach: %v", err)
	}

	// S1: two edges, last_query is the newest NON-EMPTY query (the drill's
	// empty query is filtered out, so "jwt expiry" wins even though the drill
	// is newer).
	s1, ok := reach["S1"]
	if !ok {
		t.Fatal("S1 missing from reach")
	}
	if s1.Count != 2 {
		t.Errorf("S1 count = %d, want 2", s1.Count)
	}
	if s1.Query != "jwt expiry" {
		t.Errorf("S1 query = %q, want %q", s1.Query, "jwt expiry")
	}
	if !s1.LastTS.Equal(t2) {
		t.Errorf("S1 last_ts = %v, want %v", s1.LastTS, t2)
	}

	// S2: one edge.
	if reach["S2"].Count != 1 || reach["S2"].Query != "token refresh" {
		t.Errorf("S2 = %+v, want count 1 query 'token refresh'", reach["S2"])
	}

	// S3: never reached — absent.
	if _, ok := reach["S3"]; ok {
		t.Error("S3 should be absent from reach map")
	}
}

// TestMigrateDataSchema_CreatesRecallEdges guards the backwards-compat path: a
// data.db written by an older rekal (before recall_edges existed) only runs
// migrations on open, never the full dataDDL, so the table must be ensured in
// MigrateDataSchema — not just dataDDL — or InsertRecallEdges fails with
// "table recall_edges does not exist" on every existing store.
func TestMigrateDataSchema_CreatesRecallEdges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close() //nolint:errcheck
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}
	// Simulate an old store that predates the table.
	if _, err := d.Exec(`DROP TABLE IF EXISTS recall_edges`); err != nil {
		t.Fatalf("drop recall_edges: %v", err)
	}
	// Migration runs on every open — it must recreate the table in place.
	if err := MigrateDataSchema(d); err != nil {
		t.Fatalf("MigrateDataSchema: %v", err)
	}
	if err := InsertRecallEdges(d, []RecallEdge{
		{ID: "e1", TS: time.Now().UTC(), Kind: "drill", Target: "S1"},
	}); err != nil {
		t.Fatalf("recall_edges not created by migration: %v", err)
	}
}

func TestLoadReachEmptyOrMissingTable(t *testing.T) {
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
	// No InitIndexSchema, no session_reach table — LoadReach must degrade to an
	// empty map, not error (old index built before the feature).
	reach, err := LoadReach(indexDB, []string{"S1"})
	if err != nil {
		t.Fatalf("LoadReach on missing table errored: %v", err)
	}
	if len(reach) != 0 {
		t.Fatalf("LoadReach on missing table = %v, want empty", reach)
	}
	// Empty id list is a no-op.
	if r, err := LoadReach(indexDB, nil); err != nil || len(r) != 0 {
		t.Fatalf("LoadReach(nil) = %v, %v", r, err)
	}
}
