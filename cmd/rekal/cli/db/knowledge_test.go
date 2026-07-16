package db

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestKnowledgeEmbeddings covers the semantic rung's storage helpers: the
// missing-vectors join that drives budgeted convergence, the store/query
// round trip (content-hash keyed, conflict-ignoring), and orphan pruning
// after chunks are deleted or rewritten.
func TestKnowledgeEmbeddings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := EnsureKnowledgeSchema(d); err != nil {
		t.Fatalf("EnsureKnowledgeSchema: %v", err)
	}

	if err := InsertKnowledgeChunks(d, []KnowledgeChunkRow{
		{ID: "a.md#1", Path: "a.md", StartLine: 1, EndLine: 2, Content: "alpha", ContentHash: "h-a", BlobSHA: "b1"},
		{ID: "b.md#1", Path: "b.md", StartLine: 1, EndLine: 2, Content: "beta", ContentHash: "h-b", BlobSHA: "b2"},
		// Duplicate content in a second file — one hash, one embedding.
		{ID: "c.md#1", Path: "c.md", StartLine: 1, EndLine: 2, Content: "alpha", ContentHash: "h-a", BlobSHA: "b3"},
	}); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	// All hashes are missing; distinct by hash (2, not 3).
	missing, err := QueryKnowledgeChunksMissingEmbeddings(d, "m1", 0)
	if err != nil {
		t.Fatalf("missing query: %v", err)
	}
	if len(missing) != 2 || missing["h-a"] != "alpha" || missing["h-b"] != "beta" {
		t.Fatalf("missing = %+v, want h-a/h-b", missing)
	}

	// Limit bounds one budgeted pass.
	limited, err := QueryKnowledgeChunksMissingEmbeddings(d, "m1", 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("limited missing = %+v (err %v), want exactly 1", limited, err)
	}

	// Store one vector; the join converges to the remaining hash.
	if err := StoreKnowledgeEmbeddings(d, map[string][]float64{"h-a": {0.1, 0.2}}, "m1"); err != nil {
		t.Fatalf("store: %v", err)
	}
	missing, err = QueryKnowledgeChunksMissingEmbeddings(d, "m1", 0)
	if err != nil || len(missing) != 1 || missing["h-b"] != "beta" {
		t.Fatalf("after store, missing = %+v (err %v), want just h-b", missing, err)
	}

	// Re-store is a no-op (ON CONFLICT DO NOTHING), and round trip works.
	if err := StoreKnowledgeEmbeddings(d, map[string][]float64{"h-a": {9, 9}}, "m1"); err != nil {
		t.Fatalf("re-store: %v", err)
	}
	got, err := QueryKnowledgeEmbeddings(d, "m1")
	if err != nil {
		t.Fatalf("query embeddings: %v", err)
	}
	// FLOAT[] stores float32 — compare with tolerance, and confirm the
	// re-store didn't overwrite (9,9 would fail this bound).
	if len(got) != 1 || len(got["h-a"]) != 2 || math.Abs(got["h-a"][0]-0.1) > 1e-6 {
		t.Fatalf("round trip = %+v, want original h-a vector", got)
	}

	// Model keying: another model sees everything as missing.
	missing, err = QueryKnowledgeChunksMissingEmbeddings(d, "m2", 0)
	if err != nil || len(missing) != 2 {
		t.Fatalf("m2 missing = %+v (err %v), want 2", missing, err)
	}

	// Prune: drop every chunk with hash h-a; its vector goes with it.
	if err := DeleteKnowledgeChunks(d, []string{"a.md", "c.md"}); err != nil {
		t.Fatalf("delete chunks: %v", err)
	}
	if err := PruneKnowledgeEmbeddings(d); err != nil {
		t.Fatalf("prune: %v", err)
	}
	got, err = QueryKnowledgeEmbeddings(d, "m1")
	if err != nil || len(got) != 0 {
		t.Fatalf("after prune, embeddings = %+v (err %v), want empty", got, err)
	}
}
