package search

import (
	"database/sql"
	"testing"
)

// TestLoadSessionFacetsBatch_MatchesPerRowLookup is the regression test for
// the buildResults N+1 fix: batch-loading facets for several candidate
// sessions at once must return exactly the same data a per-session lookup
// would, for every candidate, plus correctly omit sessions absent from
// session_facets.
func TestLoadSessionFacetsBatch_MatchesPerRowLookup(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	seedFacet(t, indexDB, "s1", "alice@example.com", "human")
	seedFacet(t, indexDB, "s2", "bob@example.com", "agent")
	seedFacet(t, indexDB, "s3", "", "human")

	got, err := loadSessionFacetsBatch(indexDB, []string{"s1", "s2", "s3", "missing"})
	if err != nil {
		t.Fatalf("loadSessionFacetsBatch: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d facets, want 3 (missing session must be absent)", len(got))
	}
	if _, ok := got["missing"]; ok {
		t.Error("expected no entry for a session absent from session_facets")
	}
	if got["s1"].actorType != "human" || nullStr(got["s1"].email) != "alice@example.com" {
		t.Errorf("s1 facet = %+v, want actorType=human email=alice@example.com", got["s1"])
	}
	if got["s2"].actorType != "agent" || nullStr(got["s2"].email) != "bob@example.com" {
		t.Errorf("s2 facet = %+v, want actorType=agent email=bob@example.com", got["s2"])
	}
}

func TestLoadSessionFacetsBatch_EmptyInput(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)
	got, err := loadSessionFacetsBatch(indexDB, nil)
	if err != nil {
		t.Fatalf("loadSessionFacetsBatch(nil): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map for empty input, got %d entries", len(got))
	}
}

// TestLoadSessionFilesBatch_MatchesPerRowLookup is the files_index
// equivalent of the facets test above.
func TestLoadSessionFilesBatch_MatchesPerRowLookup(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	if _, err := indexDB.Exec(
		`INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type) VALUES
		 ('cp1', 's1', 'a.go', 'M'),
		 ('cp1', 's1', 'b.go', 'A'),
		 ('cp1', 's2', 'c.go', 'M')`,
	); err != nil {
		t.Fatalf("seed files_index: %v", err)
	}

	got, err := loadSessionFilesBatch(indexDB, []string{"s1", "s2", "s3"})
	if err != nil {
		t.Fatalf("loadSessionFilesBatch: %v", err)
	}
	if len(got["s1"]) != 2 {
		t.Errorf("s1 files = %v, want 2 entries", got["s1"])
	}
	if len(got["s2"]) != 1 || got["s2"][0] != "c.go" {
		t.Errorf("s2 files = %v, want [c.go]", got["s2"])
	}
	if _, ok := got["s3"]; ok {
		t.Error("expected no entry for a session with no files touched")
	}
}

// seedFacet inserts a minimal session_facets row for batch-loading tests.
func seedFacet(t *testing.T, indexDB *sql.DB, sessionID, email, actorType string) {
	t.Helper()
	var emailVal interface{}
	if email != "" {
		emailVal = email
	}
	_, err := indexDB.Exec(
		`INSERT INTO session_facets (session_id, user_email, actor_type, captured_at) VALUES ($1, $2, $3, '2026-07-03 10:00:00')`,
		sessionID, emailVal, actorType,
	)
	if err != nil {
		t.Fatalf("seed session_facets %s: %v", sessionID, err)
	}
}
