package search

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// TestKnowledgeLayer covers the knowledge layer end-to-end against a real FTS
// index: chunk hits aggregate to file-level pointers (best chunk drives
// anchor/snippet, runners-up become Also, coverage earns a bonus), the
// provenance edge joins files_index, and the sessions ranking is untouched by
// the block's presence.
func TestKnowledgeLayer(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("ensure knowledge schema: %v", err)
	}

	// docs/auth.md matches "token" in two sections; docs/deploy.md once.
	if err := db.InsertKnowledgeChunks(indexDB, []db.KnowledgeChunkRow{
		{
			ID: "docs/auth.md#11", Path: "docs/auth.md",
			Anchor: "### Refresh rotation", Breadcrumb: "Auth Guide > Token handling > Refresh rotation",
			StartLine: 11, EndLine: 13,
			Content:     "Auth Guide > Token handling > Refresh rotation\n\nRefresh token rotates on every use.",
			ContentHash: "h1", BlobSHA: "b1",
		},
		{
			ID: "docs/auth.md#15", Path: "docs/auth.md",
			Anchor: "## Failure modes", Breadcrumb: "Auth Guide > Failure modes",
			StartLine: 15, EndLine: 17,
			Content:     "Auth Guide > Failure modes\n\nAn expired token returns 401.",
			ContentHash: "h2", BlobSHA: "b1",
		},
		{
			ID: "docs/deploy.md#1", Path: "docs/deploy.md",
			Anchor: "# Deploy", Breadcrumb: "Deploy",
			StartLine: 1, EndLine: 3,
			Content:     "Deploy\n\nRotate the deploy token quarterly.",
			ContentHash: "h3", BlobSHA: "b2",
		},
	}); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}
	if err := db.CreateKnowledgeFTSIndex(indexDB); err != nil {
		t.Fatalf("create knowledge fts index: %v", err)
	}

	// Provenance edge: a session touched docs/auth.md.
	if _, err := indexDB.Exec(`
		INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type)
		VALUES ('cp-1', '01SESSION', 'docs/auth.md', 'M')`); err != nil {
		t.Fatalf("insert files_index: %v", err)
	}

	out, err := Run(indexDB, Filters{Query: "token"}, t.TempDir(), DefaultWeights(), stubEmbedder{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(out.Knowledge) != 2 {
		t.Fatalf("want 2 knowledge hits, got %d: %+v", len(out.Knowledge), out.Knowledge)
	}
	// Two matching sections + coverage bonus → auth.md first.
	best := out.Knowledge[0]
	if best.Path != "docs/auth.md" {
		t.Fatalf("best hit = %q, want docs/auth.md", best.Path)
	}
	if best.Anchor == "" || best.Lines == "" || best.Snippet == "" {
		t.Fatalf("hit must carry anchor+lines+snippet pointers: %+v", best)
	}
	if len(best.Also) != 1 {
		t.Fatalf("want 1 runner-up anchor, got %+v", best.Also)
	}
	if len(best.Sessions) != 1 || best.Sessions[0] != "01SESSION" {
		t.Fatalf("provenance edge = %+v, want [01SESSION]", best.Sessions)
	}
	if best.Score <= out.Knowledge[1].Score {
		t.Fatalf("coverage bonus should rank auth.md above deploy.md: %v <= %v",
			best.Score, out.Knowledge[1].Score)
	}

	// The sessions block is untouched: no turns exist, so no results.
	if len(out.Results) != 0 {
		t.Fatalf("sessions ranking must be independent of knowledge hits, got %+v", out.Results)
	}
}

// TestKnowledgeSearch_FailsSoftWithoutIndex verifies the guard chain: a
// corpus with no prose files has no knowledge table/FTS index, and recall
// carries no knowledge block instead of failing.
func TestKnowledgeSearch_FailsSoftWithoutIndex(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}

	if hits := knowledgeSearch(indexDB, "anything", t.TempDir()); hits != nil {
		t.Fatalf("knowledgeSearch without an index should be nil, got %+v", hits)
	}

	// Guarded FTS build on an empty (but existing) table is a no-op.
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("ensure knowledge schema: %v", err)
	}
	if err := db.CreateKnowledgeFTSIndex(indexDB); err != nil {
		t.Fatalf("guarded create should be a no-op, got %v", err)
	}
	if hits := knowledgeSearch(indexDB, "anything", t.TempDir()); hits != nil {
		t.Fatalf("knowledgeSearch with an empty table should be nil, got %+v", hits)
	}
}
