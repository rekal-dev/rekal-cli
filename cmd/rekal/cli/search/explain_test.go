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
