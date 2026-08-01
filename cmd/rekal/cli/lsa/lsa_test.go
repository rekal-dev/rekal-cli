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

// TestEmbed_RankDeficientCorpusDoesNotBlowUpTheQueryVector pins the numerical
// rank cutoff in Embed.
//
// The projection divides by each singular value. actualDim is capped at nDocs,
// so a store with fewer sessions than DefaultDimension factorizes to full rank
// and its trailing singular values are zero to within rounding — dividing by
// 1e-16 turns rounding error into a coefficient of ~1e16, which then owns the
// vector's norm and every cosine taken against it. Guarding on Sk[j] == 0 alone
// does not catch that: near-zero is not zero.
//
// Before the cutoff this corpus produced a query vector whose squared norm was
// 1.2e30, and every document's cosine collapsed to 0 — the LSA layer returning
// nothing at all, silently.
func TestEmbed_RankDeficientCorpusDoesNotBlowUpTheQueryVector(t *testing.T) {
	t.Parallel()

	rep := func(s string, n int) string {
		out := ""
		for i := 0; i < n; i++ {
			out += s
		}
		return out
	}
	// Two sessions per topic: a term has to appear in minTermFreq (2) sessions
	// to enter the vocabulary at all, so a corpus of one-off topics factorizes
	// to rank 1 and proves nothing.
	const base = "session commit branch repo change file line test build run "
	topics := map[string]string{
		"gate":   "export gate merged ancestor squash checkpoint wire filter predicate ",
		"duck":   "duckdb storage engine append ledger rows columns pragma attach ",
		"zstd":   "zstd dictionary compression frames body decode encode preset ",
		"nomic":  "nomic embedding daemon socket model load vectors cosine tokens ",
		"search": "search ranking bm25 hybrid weights normalize confidence digest ",
	}
	sessions := map[string]string{}
	for name, terms := range topics {
		sessions[name] = rep(base+terms, 8)
		sessions[name+"2"] = rep(base+terms, 6)
	}

	m, err := Build(sessions, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m == nil {
		t.Fatal("Build returned nil model")
	}

	q := m.Embed("how does the merged export gate work")
	var sq float64
	for _, x := range q {
		sq += x * x
	}
	if math.IsNaN(sq) || math.IsInf(sq, 0) || sq > 1e6 {
		t.Fatalf("query vector squared norm is %g — a direction below the numerical rank was inverted", sq)
	}

	// And the layer must still discriminate: the session about the gate has to
	// beat every unrelated one. A blown-up component makes them indistinguishable.
	vecs := m.Vectors()
	gateSim := CosineSimilarity(q, vecs["gate"])
	for _, id := range []string{"duck", "zstd", "nomic", "search"} {
		if got := CosineSimilarity(q, vecs[id]); got >= gateSim {
			t.Errorf("%s scores %.4f against a query about the export gate, at or above the gate session's %.4f", id, got, gateSim)
		}
	}
}
