package search

import (
	"database/sql"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// seedEqualPair builds two sessions, s-a and s-b, with identical relevance to
// the query "auth token expiry" (the same single turn), so any ranking
// difference between them comes from the recency / reach layers alone. The
// caller supplies each session's captured_at. FTS is required (skips if
// unavailable).
func seedEqualPair(t *testing.T, capturedA, capturedB string) *sql.DB {
	t.Helper()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	for _, s := range []struct{ sid, cap string }{{"s-a", capturedA}, {"s-b", capturedB}} {
		if _, err := indexDB.Exec(
			`INSERT INTO session_facets (session_id, actor_type, captured_at, turn_count, tool_call_count, file_count)
			 VALUES ($1, 'human', $2, 1, 0, 0)`, s.sid, s.cap,
		); err != nil {
			t.Fatalf("seed facet %s: %v", s.sid, err)
		}
	}
	for _, sid := range []string{"s-a", "s-b"} {
		if _, err := indexDB.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, 0, 'human', 'auth token expiry refresh', '')`, "t-"+sid, sid,
		); err != nil {
			t.Fatalf("seed turn %s: %v", sid, err)
		}
	}
	if err := db.CreateFTSIndex(indexDB); err != nil {
		t.Fatalf("CreateFTSIndex: %v", err)
	}
	return indexDB
}

// TestRecencyBoost: with recency_boost > 0 the newer of two equally-relevant
// sessions ranks first, the recency layer shows under --explain, and at boost 0
// the layer is inert.
func TestRecencyBoost(t *testing.T) {
	t.Parallel()
	indexDB := seedEqualPair(t, "2026-01-01T00:00:00Z", "2026-06-01T00:00:00Z") // s-b newer

	run := func(boost float64) []Result {
		w := DefaultWeights()
		w.RecencyBoost = boost
		out, err := Run(indexDB, Filters{Query: "auth token expiry", Explain: true}, t.TempDir(), w, stubEmbedder{})
		if err != nil {
			t.Fatalf("Run(recency_boost=%v): %v", boost, err)
		}
		return out.Results
	}

	res := run(1.0)
	if len(res) != 2 {
		t.Fatalf("want 2 results, got %d (%+v)", len(res), res)
	}
	if res[0].SessionID != "s-b" {
		t.Fatalf("recency_boost>0: want s-b (newer) first, got %s", res[0].SessionID)
	}
	if res[0].Layers == nil || res[0].Layers.Recency <= 0 {
		t.Fatalf("explain should carry a positive recency layer, got %+v", res[0].Layers)
	}

	for _, r := range run(0) {
		if r.Layers != nil && r.Layers.Recency != 0 {
			t.Fatalf("recency_boost=0: %s recency layer = %v, want 0", r.SessionID, r.Layers.Recency)
		}
	}
}

// TestReachBoost: with reach_boost > 0 the session an agent has actually
// drilled ranks ahead of an equally-relevant one (same captured_at), the reach
// layer shows under --explain, and at boost 0 the layer is inert.
func TestReachBoost(t *testing.T) {
	t.Parallel()
	indexDB := seedEqualPair(t, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z") // equal recency
	if err := db.EnsureReachSchema(indexDB); err != nil {
		t.Fatalf("ensure reach schema: %v", err)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO session_reach (target_session_id, reach_count, drill_count, last_query, last_ts)
		 VALUES ('s-b', 7, 2, 'q', '2026-01-02T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed session_reach: %v", err)
	}

	run := func(boost float64) []Result {
		w := DefaultWeights()
		w.ReachBoost = boost
		out, err := Run(indexDB, Filters{Query: "auth token expiry", Explain: true}, t.TempDir(), w, stubEmbedder{})
		if err != nil {
			t.Fatalf("Run(reach_boost=%v): %v", boost, err)
		}
		return out.Results
	}

	res := run(1.0)
	if len(res) != 2 {
		t.Fatalf("want 2 results, got %d", len(res))
	}
	if res[0].SessionID != "s-b" {
		t.Fatalf("reach_boost>0: want s-b (more reached) first, got %s", res[0].SessionID)
	}
	if res[0].Layers == nil || res[0].Layers.Reach <= 0 {
		t.Fatalf("explain should carry a positive reach layer, got %+v", res[0].Layers)
	}

	for _, r := range run(0) {
		if r.Layers != nil && r.Layers.Reach != 0 {
			t.Fatalf("reach_boost=0: %s reach layer = %v, want 0", r.SessionID, r.Layers.Reach)
		}
	}
}

// TestReachBoost_SurfacingAloneDoesNotRank pins the distinction the layer now
// makes. A recall edge says only that this engine already ranked the session
// into some window — boosting on it is the ranker rewarding its own output, and
// with 20 seeds per call it marks most of a small corpus. Only a drill, an
// agent's decision to open the session, moves ranking.
func TestReachBoost_SurfacingAloneDoesNotRank(t *testing.T) {
	t.Parallel()
	indexDB := seedEqualPair(t, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	if err := db.EnsureReachSchema(indexDB); err != nil {
		t.Fatalf("ensure reach schema: %v", err)
	}
	// Surfaced 99 times, never opened.
	if _, err := indexDB.Exec(
		`INSERT INTO session_reach (target_session_id, reach_count, drill_count, last_query, last_ts)
		 VALUES ('s-b', 99, 0, 'q', '2026-01-02T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed session_reach: %v", err)
	}

	w := DefaultWeights()
	w.ReachBoost = 1.0
	out, err := Run(indexDB, Filters{Query: "auth token expiry", Explain: true}, t.TempDir(), w, stubEmbedder{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, r := range out.Results {
		if r.Layers != nil && r.Layers.Reach != 0 {
			t.Errorf("%s: surfacing alone produced a reach layer of %v, want 0", r.SessionID, r.Layers.Reach)
		}
	}
}

// TestReachBoost_SoftFailsWithoutTable: reach_boost > 0 against an index with no
// session_reach table (older index.db) must not error — the layer fails soft.
func TestReachBoost_SoftFailsWithoutTable(t *testing.T) {
	t.Parallel()
	indexDB := seedEqualPair(t, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	// deliberately no EnsureReachSchema

	w := DefaultWeights()
	w.ReachBoost = 1.0
	out, err := Run(indexDB, Filters{Query: "auth token expiry"}, t.TempDir(), w, stubEmbedder{})
	if err != nil {
		t.Fatalf("reach_boost>0 without session_reach should fail soft, got error: %v", err)
	}
	if len(out.Results) == 0 {
		t.Fatalf("want results, got 0")
	}
}
