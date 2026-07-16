package search

import (
	"errors"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// TestLoadSessionFacet covers the single-row facet load used for synthetic
// group headers, through the shared sessionFacetCols/scan/detail path — the
// same column list every recall query uses, so a drifted column breaks here.
func TestLoadSessionFacet(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)

	if _, err := indexDB.Exec(
		`INSERT INTO session_facets (
			session_id, user_email, git_branch, actor_type, agent_id,
			captured_at, turn_count, tool_call_count, file_count,
			parent_session_id, team_name, workflow_name,
			agent_type, description, spawn_depth, origin
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		"s-1", "alice@co.com", "main", "human", nil,
		"2026-01-01T10:00:00Z", 7, 2, 1,
		nil, nil, nil, nil, nil, nil, "repo:/home/alice/api",
	); err != nil {
		t.Fatalf("insert facet: %v", err)
	}

	detail, ok := loadSessionFacet(indexDB, "s-1")
	if !ok {
		t.Fatal("loadSessionFacet: row not found")
	}
	if detail.Author != "alice@co.com" || detail.Actor != "human" || detail.Branch != "main" {
		t.Fatalf("detail mismatch: %+v", detail)
	}
	if detail.TurnCount != 7 || detail.ToolCalls != 2 {
		t.Fatalf("counts mismatch: %+v", detail)
	}
	if detail.Origin != "repo:/home/alice/api" {
		t.Fatalf("origin = %q, want repo:/home/alice/api", detail.Origin)
	}

	if _, ok := loadSessionFacet(indexDB, "missing"); ok {
		t.Fatal("loadSessionFacet returned ok for a missing session")
	}
}

// stubEmbedder keeps the deep semantic layer inert in tests: its model name
// matches no stored vectors, so nomicSearch skips the layer without loading
// the embedded model.
type stubEmbedder struct{}

func (stubEmbedder) ModelName() string                    { return "test-model-with-no-vectors" }
func (stubEmbedder) Backend() string                      { return EmbedderBackendHTTP }
func (stubEmbedder) EmbedQuery(string) ([]float64, error) { return nil, errors.New("unused") }
func (stubEmbedder) Close()                               {}

// TestFacetLayer covers the facet ranking layer end-to-end against a real
// FTS index: a session whose conversational turns never mention the query
// term is findable through its facet document when the layer is enabled
// (facet_boost > 0, the shipped default is 0.3), and invisible at an
// explicit 0 — the byte-identical-baseline off switch.
func TestFacetLayer(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}

	// Two sessions. s-tools used kubectl (tool metadata only — its turns
	// never say so); s-chat talked about databases.
	for _, ins := range []struct {
		sid, facet string
	}{
		{"s-tools", "cmd/deploy/main.go kubectl helm"},
		{"s-chat", "notes.md psql"},
	} {
		if _, err := indexDB.Exec(
			`INSERT INTO session_facets (session_id, actor_type, captured_at, turn_count, tool_call_count, file_count, facet_text)
			 VALUES ($1, 'human', '2026-01-01T10:00:00Z', 1, 1, 0, $2)`,
			ins.sid, ins.facet,
		); err != nil {
			t.Fatalf("insert facet: %v", err)
		}
	}
	if _, err := indexDB.Exec(
		`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts) VALUES
		 ('t1', 's-tools', 0, 'human', 'please roll the deployment forward', ''),
		 ('t2', 's-chat', 0, 'human', 'database connection pooling question', '')`,
	); err != nil {
		t.Fatalf("insert turns: %v", err)
	}
	if err := db.CreateFacetFTSIndex(indexDB); err != nil {
		t.Fatalf("create facet fts index: %v", err)
	}

	run := func(facetBoost float64) []Result {
		w := DefaultWeights()
		w.FacetBoost = facetBoost
		out, err := Run(indexDB, Filters{Query: "kubectl", Explain: true}, t.TempDir(), w, stubEmbedder{})
		if err != nil {
			t.Fatalf("Run(facet_boost=%v): %v", facetBoost, err)
		}
		return out.Results
	}

	// Explicit 0 (the off switch): the facet search never runs; nothing
	// matches "kubectl" because no turn mentions it.
	if results := run(0); len(results) != 0 {
		t.Fatalf("facet_boost=0: want no results, got %d (%+v)", len(results), results)
	}

	// Enabled (the shipped default): the facet layer surfaces s-tools, with
	// the layer's score visible under --explain.
	results := run(DefaultWeights().FacetBoost)
	if len(results) != 1 || results[0].SessionID != "s-tools" {
		t.Fatalf("facet_boost=0.3: want [s-tools], got %+v", results)
	}
	if results[0].Layers == nil || results[0].Layers.Facet <= 0 {
		t.Fatalf("explain should carry a positive facet layer score, got %+v", results[0].Layers)
	}
}

// TestCreateFacetFTSIndex_GuardedWhenEmpty verifies the guard: with no facet
// material the index build is a no-op (no error), and facetSearch reports the
// layer off (nil) instead of failing recall.
func TestCreateFacetFTSIndex_GuardedWhenEmpty(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	if err := db.CreateFacetFTSIndex(indexDB); err != nil {
		t.Fatalf("guarded create should be a no-op, got %v", err)
	}
	if scores := facetSearch(indexDB, "anything"); scores != nil {
		t.Fatalf("facetSearch without an index should be nil, got %v", scores)
	}
}
