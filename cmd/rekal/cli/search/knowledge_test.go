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

	if hits := knowledgeSearch(indexDB, "anything", t.TempDir(), DefaultWeights(), nil, nil, ""); hits != nil {
		t.Fatalf("knowledgeSearch without an index should be nil, got %+v", hits)
	}

	// Guarded FTS build on an empty (but existing) table is a no-op.
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("ensure knowledge schema: %v", err)
	}
	if err := db.CreateKnowledgeFTSIndex(indexDB); err != nil {
		t.Fatalf("guarded create should be a no-op, got %v", err)
	}
	if hits := knowledgeSearch(indexDB, "anything", t.TempDir(), DefaultWeights(), nil, nil, ""); hits != nil {
		t.Fatalf("knowledgeSearch with an empty table should be nil, got %+v", hits)
	}
}

// TestKnowledgeSemanticLayer covers the semantic rung: cosine re-ranks BM25
// candidates (never a full-corpus scan), BM25 and semantic scores blend with
// the session ranking's keyword/semantic split, and a missing vector store
// degrades to keyword-only rather than failing.
func TestKnowledgeSemanticLayer(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("ensure knowledge schema: %v", err)
	}

	// Both docs match "token" by keyword. deploy.md's vector aligns with the
	// query; auth.md's does not — semantic re-rank should promote deploy.
	// orphan.md has a strong vector but zero keyword overlap and must stay
	// out (no full-corpus semantic-only pass).
	if err := db.InsertKnowledgeChunks(indexDB, []db.KnowledgeChunkRow{
		{
			ID: "docs/auth.md#1", Path: "docs/auth.md",
			Anchor: "# Auth", Breadcrumb: "Auth",
			StartLine: 1, EndLine: 3,
			Content:     "Auth\n\nThe token rotates on every use.",
			ContentHash: "hash-auth", BlobSHA: "b1",
		},
		{
			ID: "docs/deploy.md#1", Path: "docs/deploy.md",
			Anchor: "# Deploy", Breadcrumb: "Deploy",
			StartLine: 1, EndLine: 3,
			Content:     "Deploy\n\nRotate the deploy token quarterly.",
			ContentHash: "hash-deploy", BlobSHA: "b2",
		},
		{
			ID: "docs/orphan.md#1", Path: "docs/orphan.md",
			Anchor: "# Orphan", Breadcrumb: "Orphan",
			StartLine: 1, EndLine: 3,
			Content:     "Orphan\n\nCredentials expire after an hour of inactivity.",
			ContentHash: "hash-orphan", BlobSHA: "b3",
		},
	}); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}
	if err := db.CreateKnowledgeFTSIndex(indexDB); err != nil {
		t.Fatalf("create knowledge fts index: %v", err)
	}
	if err := db.StoreKnowledgeEmbeddings(indexDB, map[string][]float64{
		"hash-auth":   {0, 1, 0},
		"hash-deploy": {1, 0, 0},
		"hash-orphan": {1, 0, 0},
	}, "test-model"); err != nil {
		t.Fatalf("store knowledge embeddings: %v", err)
	}

	queryVec := []float64{1, 0, 0}
	hits := knowledgeSearch(indexDB, "token", t.TempDir(), DefaultWeights(), nil, queryVec, "test-model")
	if len(hits) != 2 {
		t.Fatalf("want 2 BM25 candidate hits, got %+v", hits)
	}
	for _, h := range hits {
		if h.Path == "docs/orphan.md" {
			t.Fatalf("semantic-only (no BM25) chunk must not appear: %+v", hits)
		}
	}
	if hits[0].Path != "docs/deploy.md" {
		t.Fatalf("semantic-aligned BM25 candidate should rank first, got %+v", hits)
	}

	// Unknown model → no vectors → keyword-only ranking among BM25 hits.
	kw := knowledgeSearch(indexDB, "token", t.TempDir(), DefaultWeights(), nil, queryVec, "other-model")
	if len(kw) != 2 {
		t.Fatalf("keyword-only should return both BM25 hits, got %+v", kw)
	}
	if kw[0].Score != 1 {
		t.Fatalf("keyword-only top score should be normalized BM25 (1.0), got %v", kw[0].Score)
	}
}
