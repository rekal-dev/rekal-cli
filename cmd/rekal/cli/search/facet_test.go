package search

import (
	"testing"
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
