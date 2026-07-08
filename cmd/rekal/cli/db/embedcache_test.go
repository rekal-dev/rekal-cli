package db

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbedCache_RoundTrip(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gitRoot, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	cache, err := OpenEmbedCache(gitRoot)
	if err != nil {
		t.Fatalf("OpenEmbedCache: %v", err)
	}
	defer cache.Close()

	put := map[string][]float64{
		"hash-a": {0.1, 0.2, 0.3},
		"hash-b": {1, 2, 3},
	}
	if err := CachePutEmbeddings(cache, "model-x", put); err != nil {
		t.Fatalf("CachePutEmbeddings: %v", err)
	}
	// Idempotent re-put of the same key must not error.
	if err := CachePutEmbeddings(cache, "model-x", map[string][]float64{"hash-a": {0.1, 0.2, 0.3}}); err != nil {
		t.Fatalf("re-put: %v", err)
	}

	got, err := CacheGetEmbeddings(cache, "model-x", []string{"hash-a", "hash-b", "hash-missing"})
	if err != nil {
		t.Fatalf("CacheGetEmbeddings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2 (missing hash absent, not error)", len(got))
	}
	// DuckDB FLOAT[] is 32-bit (same as session_embeddings) — compare with
	// float32 tolerance.
	if v := got["hash-a"]; len(v) != 3 || math.Abs(v[1]-0.2) > 1e-6 {
		t.Fatalf("hash-a = %v", v)
	}

	// Model isolation: another model sees nothing — this is exactly how a
	// model switch invalidates the cache without any invalidation code.
	other, err := CacheGetEmbeddings(cache, "model-y", []string{"hash-a"})
	if err != nil {
		t.Fatalf("CacheGetEmbeddings other model: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("model-y sees %d entries, want 0", len(other))
	}
}
