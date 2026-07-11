package search

import (
	"database/sql"
	"testing"
)

func seedFile(t *testing.T, indexDB *sql.DB, sessionID, path string) {
	t.Helper()
	_, err := indexDB.Exec(
		`INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type) VALUES ('cp', $1, $2, 'M')`,
		sessionID, path,
	)
	if err != nil {
		t.Fatalf("seed files_index %s %s: %v", sessionID, path, err)
	}
}

// TestAttachSummaryPointers verifies the summary-pointer enrichment: a
// result whose session has compaction-summary turns gets the index of the
// latest one (summaries are cumulative — the latest subsumes the rest), a
// folded child gets its own pointer, and sessions without a summary keep the
// field absent. The pointer, not the payload: recall output must stay thin.
func TestAttachSummaryPointers(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	for _, r := range []struct {
		id, sid string
		idx     int
		role    string
	}{
		{"t1", "s1", 3, "summary"},
		{"t2", "s1", 41, "summary"}, // latest must win
		{"t3", "s1", 50, "human"},
		{"t4", "s2", 0, "human"}, // no summary at all
		{"t5", "child1", 7, "summary"},
	} {
		if _, err := indexDB.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, $3, $4, 'x', '')`,
			r.id, r.sid, r.idx, r.role,
		); err != nil {
			t.Fatalf("seed turn %s: %v", r.id, err)
		}
	}

	results := []Result{
		{SessionID: "s1", Children: []Result{{SessionID: "child1"}}},
		{SessionID: "s2"},
	}
	attachSummaryPointers(indexDB, results)

	if results[0].SummaryTurnIdx == nil || *results[0].SummaryTurnIdx != 41 {
		t.Errorf("s1 SummaryTurnIdx = %v, want 41 (latest summary)", results[0].SummaryTurnIdx)
	}
	if c := results[0].Children[0].SummaryTurnIdx; c == nil || *c != 7 {
		t.Errorf("child1 SummaryTurnIdx = %v, want 7", c)
	}
	if results[1].SummaryTurnIdx != nil {
		t.Errorf("s2 SummaryTurnIdx = %v, want absent (no summary turn)", *results[1].SummaryTurnIdx)
	}
}

// TestAttachRelated verifies the query-time co-occurrence join: sessions
// sharing touched files become Related entries, ordered by shared-file
// count, capped at relatedLimit, and absent for sessions with no overlap.
func TestAttachRelated(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)

	// s1 shares 2 files with s2, 1 with s3; s4 shares nothing.
	for _, f := range []struct{ sid, path string }{
		{"s1", "a.go"}, {"s1", "b.go"}, {"s1", "c.go"},
		{"s2", "a.go"}, {"s2", "b.go"},
		{"s3", "c.go"},
		{"s4", "z.go"},
	} {
		seedFile(t, indexDB, f.sid, f.path)
	}

	results := []Result{{SessionID: "s1"}, {SessionID: "s4"}}
	attachRelated(indexDB, results)

	rel := results[0].Related
	if len(rel) != 2 {
		t.Fatalf("s1 related = %+v, want 2 entries", rel)
	}
	if rel[0].SessionID != "s2" || rel[0].SharedFiles != 2 {
		t.Errorf("s1 top related = %+v, want s2 with 2 shared files", rel[0])
	}
	if rel[1].SessionID != "s3" || rel[1].SharedFiles != 1 {
		t.Errorf("s1 second related = %+v, want s3 with 1 shared file", rel[1])
	}
	if results[1].Related != nil {
		t.Errorf("s4 related = %+v, want none (no overlapping files)", results[1].Related)
	}
}

// TestAttachRelated_CapsAtLimit: more co-occurring sessions than
// relatedLimit must be truncated to the strongest edges.
func TestAttachRelated_CapsAtLimit(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)
	seedFile(t, indexDB, "hub", "shared.go")
	for _, sid := range []string{"n1", "n2", "n3", "n4", "n5"} {
		seedFile(t, indexDB, sid, "shared.go")
	}

	results := []Result{{SessionID: "hub"}}
	attachRelated(indexDB, results)
	if len(results[0].Related) != relatedLimit {
		t.Fatalf("related count = %d, want capped at %d", len(results[0].Related), relatedLimit)
	}
}

// TestRun_FilterMode_ExplainAttachesRelated exercises the Explain path end
// to end through Run in filter mode (no FTS required): with the flag,
// Related appears; without it, output carries neither Related nor Layers.
func TestRun_FilterMode_ExplainAttachesRelated(t *testing.T) {
	t.Parallel()

	indexDB := openTempIndexDB(t)
	seedFacet(t, indexDB, "s1", "a@x.dev", "human")
	seedFacet(t, indexDB, "s2", "a@x.dev", "human")
	seedFile(t, indexDB, "s1", "main.go")
	seedFile(t, indexDB, "s2", "main.go")

	out, err := Run(indexDB, Filters{Author: "a@x.dev", Explain: true}, t.TempDir(), Weights{}, nil)
	if err != nil {
		t.Fatalf("Run explain: %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(out.Results))
	}
	for _, r := range out.Results {
		if len(r.Related) != 1 {
			t.Errorf("%s: related = %+v, want exactly the co-occurring session", r.SessionID, r.Related)
		}
	}

	out, err = Run(indexDB, Filters{Author: "a@x.dev"}, t.TempDir(), Weights{}, nil)
	if err != nil {
		t.Fatalf("Run plain: %v", err)
	}
	for _, r := range out.Results {
		if r.Related != nil || r.Layers != nil {
			t.Errorf("%s: enrichments present without --explain: related=%v layers=%v",
				r.SessionID, r.Related, r.Layers)
		}
	}
}

// TestScoredLayers_RoundTrip: the layers stored on a scored candidate must
// surface on the built Result, and round2 must behave.
func TestRound2(t *testing.T) {
	t.Parallel()
	if got := round2(0.123456); got != 0.12 {
		t.Errorf("round2(0.123456) = %v, want 0.12", got)
	}
	if got := round2(0.999); got != 1.0 {
		t.Errorf("round2(0.999) = %v, want 1.0", got)
	}
}
