package search

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// TestNomicSearch_ModelMismatchReportsReason covers the silent-skip bug:
// index built with Cohere (or any non-query model) + query embedder for a
// different model must return an error that hybridSearch records in
// skipped["nomic"], not a quiet empty score map.
func TestNomicSearch_ModelMismatchReportsReason(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)

	if err := db.StoreEmbeddings(indexDB, map[string][]float64{
		"s1": {0.1, 0.2, 0.3},
	}, "@bedrock-au/cohere.embed-english-v3"); err != nil {
		t.Fatalf("StoreEmbeddings: %v", err)
	}
	if err := db.WriteIndexState(indexDB, "embed_model", "@bedrock-au/cohere.embed-english-v3"); err != nil {
		t.Fatalf("WriteIndexState: %v", err)
	}

	scores, _, _, err := nomicSearch(indexDB, "auth", t.TempDir(), stubEmbedder{})
	if err == nil {
		t.Fatal("expected model-mismatch error, got nil")
	}
	if scores != nil {
		t.Fatalf("scores = %v, want nil", scores)
	}
	if !strings.Contains(err.Error(), "no vectors for query model") {
		t.Fatalf("err = %v, want no vectors for query model", err)
	}
	if !strings.Contains(err.Error(), "@bedrock-au/cohere.embed-english-v3") {
		t.Fatalf("err = %v, want index model listed", err)
	}
}

// TestNomicSearch_NoSemanticVectorsIsQuietSkip: an index with only LSA (or
// nothing) legitimately has no deep semantic layer — no error.
func TestNomicSearch_NoSemanticVectorsIsQuietSkip(t *testing.T) {
	t.Parallel()
	indexDB := openTempIndexDB(t)
	if err := db.StoreEmbeddings(indexDB, map[string][]float64{
		"s1": {0.1, 0.2},
	}, "lsa-v1"); err != nil {
		t.Fatalf("StoreEmbeddings: %v", err)
	}

	scores, _, _, err := nomicSearch(indexDB, "auth", t.TempDir(), stubEmbedder{})
	if err != nil {
		t.Fatalf("err = %v, want nil (no semantic models)", err)
	}
	if scores != nil {
		t.Fatalf("scores = %v, want nil", scores)
	}
}

// TestLineage_RecordsNomicMismatchSkip ensures the mismatch surfaces in the
// result event's skipped map so scoring_lineage explains the neural miss.
func TestLineage_RecordsNomicMismatchSkip(t *testing.T) {
	t.Parallel()
	indexDB := seedLineageCorpus(t)
	if err := db.StoreEmbeddings(indexDB, map[string][]float64{
		"s-auth": {0.1, 0.2, 0.3},
	}, "@bedrock-au/cohere.embed-english-v3"); err != nil {
		t.Fatalf("StoreEmbeddings: %v", err)
	}

	var buf bytes.Buffer
	lin := NewNDJSONLineage(&buf, 5)
	_, err := Run(indexDB, Filters{Query: "auth", Lineage: lin}, t.TempDir(), DefaultWeights(), stubEmbedder{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	lin.FlushResult(0)

	var sawSkip bool
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var head struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal([]byte(line), &head); err != nil {
			t.Fatalf("ndjson: %v", err)
		}
		if head.Event != "result" {
			continue
		}
		var r struct {
			Skipped map[string]string `json:"skipped"`
		}
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("result: %v", err)
		}
		if reason, ok := r.Skipped["nomic"]; ok && strings.Contains(reason, "no vectors for query model") {
			sawSkip = true
			if !strings.Contains(reason, "cohere") {
				t.Fatalf("skip reason missing index model: %s", reason)
			}
		}
	}
	if !sawSkip {
		t.Fatalf("expected skipped.nomic mismatch in result; log=%s", buf.String())
	}
}
