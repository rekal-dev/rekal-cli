package search

import (
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
)

func openTempIndexDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexDB, err := db.OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		indexDB.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { indexDB.Close() })
	return indexDB
}

// TestLSAProjection_StoreLoadRoundTrip verifies the actual persistence path
// (base64-in-index_state, not just the lsa package's gob round trip): store
// a fitted model's projection state, reload it in a fresh call, and confirm
// Embed produces the same result as the original model.
func TestLSAProjection_StoreLoadRoundTrip(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	sessions := map[string]string{
		"s1": "JWT authentication token expiry refresh login security middleware",
		"s2": "JWT token validation auth middleware bearer header claims expiry",
		"s3": "database connection pooling query optimization index performance SQL",
		"s4": "database schema migration table column index query performance tuning",
	}
	model, err := lsa.Build(sessions, 3)
	if err != nil || model == nil {
		t.Fatalf("lsa.Build: model=%v err=%v", model, err)
	}

	if err := StoreLSAProjection(indexDB, model); err != nil {
		t.Fatalf("StoreLSAProjection: %v", err)
	}

	loaded, err := loadLSAProjection(indexDB)
	if err != nil {
		t.Fatalf("loadLSAProjection: %v", err)
	}
	if loaded == nil {
		t.Fatal("loadLSAProjection returned nil, nil after storing a projection")
	}

	for _, query := range []string{"JWT authentication", "database performance"} {
		want := model.Embed(query)
		got := loaded.Embed(query)
		if len(got) != len(want) {
			t.Fatalf("Embed(%q): len = %d, want %d", query, len(got), len(want))
		}
		for i := range want {
			if math.Abs(got[i]-want[i]) > 1e-9 {
				t.Errorf("Embed(%q)[%d] = %v, want %v", query, i, got[i], want[i])
			}
		}
	}
}

// TestLoadLSAProjection_AbsentReturnsNil covers a fresh index.db (or one
// built before this cache existed) — recall must fall back to rebuilding,
// not error out.
func TestLoadLSAProjection_AbsentReturnsNil(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	model, err := loadLSAProjection(indexDB)
	if err != nil {
		t.Fatalf("loadLSAProjection: %v", err)
	}
	if model != nil {
		t.Errorf("expected nil model for an index.db with no stored projection, got %+v", model)
	}
}

// TestStoreLSAProjection_NilModelIsNoop matches StoreLSAProjection's doc: a
// nil model (LSA build failed/skipped) must not error and must not write a
// stale/garbage entry.
func TestStoreLSAProjection_NilModelIsNoop(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	if err := StoreLSAProjection(indexDB, nil); err != nil {
		t.Fatalf("StoreLSAProjection(nil): %v", err)
	}
	model, err := loadLSAProjection(indexDB)
	if err != nil {
		t.Fatalf("loadLSAProjection: %v", err)
	}
	if model != nil {
		t.Error("expected no projection stored after StoreLSAProjection(nil)")
	}
}
