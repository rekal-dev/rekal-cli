package lsa

import (
	"math"
	"testing"
)

func TestTokenize_Basic(t *testing.T) {
	t.Parallel()
	tokens := Tokenize("Hello World! This is a test.")
	// "this", "is", "a" are stopwords; "hello", "world", "test" remain
	found := make(map[string]bool)
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["hello"] {
		t.Error("expected 'hello' in tokens")
	}
	if !found["world"] {
		t.Error("expected 'world' in tokens")
	}
	if !found["test"] {
		t.Error("expected 'test' in tokens")
	}
	if found["this"] {
		t.Error("'this' should be filtered as stopword")
	}
	if found["is"] {
		t.Error("'is' should be filtered as stopword")
	}
}

func TestTokenize_Empty(t *testing.T) {
	t.Parallel()
	tokens := Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected empty tokens, got %v", tokens)
	}
}

func TestTokenize_Numbers(t *testing.T) {
	t.Parallel()
	tokens := Tokenize("error 404 not found")
	found := make(map[string]bool)
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["error"] {
		t.Error("expected 'error' in tokens")
	}
	if !found["404"] {
		t.Error("expected '404' in tokens")
	}
	if !found["found"] {
		t.Error("expected 'found' in tokens")
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	t.Parallel()
	a := []float64{1, 2, 3}
	sim := CosineSimilarity(a, a)
	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("expected 1.0, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	t.Parallel()
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	sim := CosineSimilarity(a, b)
	if math.Abs(sim) > 1e-9 {
		t.Errorf("expected 0.0, got %f", sim)
	}
}

func TestCosineSimilarity_Zero(t *testing.T) {
	t.Parallel()
	a := []float64{0, 0, 0}
	b := []float64{1, 2, 3}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0.0, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	t.Parallel()
	a := []float64{1, 2}
	b := []float64{1, 2, 3}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0.0 for different lengths, got %f", sim)
	}
}

func TestBuild_TooFewSessions(t *testing.T) {
	t.Parallel()
	sessions := map[string]string{
		"s1": "hello world",
	}
	model, err := Build(sessions, DefaultDimension)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != nil {
		t.Error("expected nil model for single session")
	}
}

func TestBuild_And_Embed(t *testing.T) {
	t.Parallel()
	sessions := map[string]string{
		"s1": "JWT authentication token expiry refresh login security middleware",
		"s2": "JWT token validation auth middleware bearer header claims expiry",
		"s3": "database connection pooling query optimization index performance SQL",
		"s4": "database schema migration table column index query performance tuning",
	}

	model, err := Build(sessions, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.Dim != 3 {
		t.Errorf("expected dim 3, got %d", model.Dim)
	}

	// Vectors should have entries for all sessions.
	vectors := model.Vectors()
	if len(vectors) != 4 {
		t.Errorf("expected 4 vectors, got %d", len(vectors))
	}
	for _, id := range []string{"s1", "s2", "s3", "s4"} {
		if _, ok := vectors[id]; !ok {
			t.Errorf("missing vector for %s", id)
		}
	}

	// Query about JWT should be more similar to s1/s2 than s3/s4.
	queryVec := model.Embed("JWT authentication")
	simS1 := CosineSimilarity(queryVec, vectors["s1"])
	simS2 := CosineSimilarity(queryVec, vectors["s2"])
	simS3 := CosineSimilarity(queryVec, vectors["s3"])
	simS4 := CosineSimilarity(queryVec, vectors["s4"])

	authAvg := (simS1 + simS2) / 2
	dbAvg := (simS3 + simS4) / 2

	if authAvg <= dbAvg {
		t.Errorf("expected auth sessions to be more similar to JWT query: auth_avg=%f, db_avg=%f", authAvg, dbAvg)
	}
}

func TestBuild_EmptySessions(t *testing.T) {
	t.Parallel()
	sessions := map[string]string{}
	model, err := Build(sessions, DefaultDimension)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != nil {
		t.Error("expected nil model for empty sessions")
	}
}

func TestSimpleStem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, expected string
	}{
		{"running", "runn"},
		{"authentication", "authentica"},
		{"connections", "connection"},
		{"go", "go"}, // too short to stem
	}
	for _, tc := range cases {
		got := simpleStem(tc.input)
		if got != tc.expected {
			t.Errorf("simpleStem(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
