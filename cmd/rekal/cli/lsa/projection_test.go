package lsa

import (
	"math"
	"testing"
)

// TestEncodeDecodeProjection_EmbedMatchesOriginal is the core regression
// guard for the recall perf fix: a query embedded against a decoded
// projection must produce the same result as embedding it against the
// freshly-built model, since recall.go now prefers the persisted
// projection over rebuilding from raw content.
func TestEncodeDecodeProjection_EmbedMatchesOriginal(t *testing.T) {
	t.Parallel()

	sessions := map[string]string{
		"s1": "JWT authentication token expiry refresh login security middleware",
		"s2": "JWT token validation auth middleware bearer header claims expiry",
		"s3": "database connection pooling query optimization index performance SQL",
		"s4": "database schema migration table column index query performance tuning",
	}

	model, err := Build(sessions, 3)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}

	encoded, err := EncodeProjection(model)
	if err != nil {
		t.Fatalf("EncodeProjection: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("EncodeProjection returned empty bytes")
	}

	decoded, err := DecodeProjection(encoded)
	if err != nil {
		t.Fatalf("DecodeProjection: %v", err)
	}
	if decoded.Dim != model.Dim {
		t.Errorf("decoded Dim = %d, want %d", decoded.Dim, model.Dim)
	}
	if len(decoded.Vocabulary) != len(model.Vocabulary) {
		t.Errorf("decoded Vocabulary size = %d, want %d", len(decoded.Vocabulary), len(model.Vocabulary))
	}

	for _, query := range []string{"JWT authentication", "database performance tuning", ""} {
		want := model.Embed(query)
		got := decoded.Embed(query)
		if len(got) != len(want) {
			t.Fatalf("Embed(%q): decoded len = %d, want %d", query, len(got), len(want))
		}
		for i := range want {
			if math.Abs(got[i]-want[i]) > 1e-9 {
				t.Errorf("Embed(%q)[%d] = %v, want %v", query, i, got[i], want[i])
			}
		}
	}
}

func TestDecodeProjection_RejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, err := DecodeProjection([]byte("not a gob stream")); err == nil {
		t.Error("expected an error decoding garbage bytes, got nil")
	}
}

func TestEncodeProjection_RejectsNilModel(t *testing.T) {
	t.Parallel()

	if _, err := EncodeProjection(nil); err == nil {
		t.Error("expected an error encoding a nil model, got nil")
	}
}
