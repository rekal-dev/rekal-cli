package embedhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer implements just enough of the OpenAI embeddings API: each input
// string maps to a deterministic 3-dim vector so tests can verify routing.
func fakeServer(t *testing.T, requests *atomic.Int64, wantAuth string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/embeddings" {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if wantAuth != "" && r.Header.Get("Authorization") != wantAuth {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := embedResponse{}
		for i, in := range req.Input {
			resp.Data = append(resp.Data, struct {
				Index     int       `json:"index"`
				Embedding []float64 `json:"embedding"`
			}{Index: i, Embedding: []float64{float64(len(in)), float64(i), 1}})
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestEmbedQuery_PrefixAndAuth(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	srv := fakeServer(t, &requests, "Bearer sekrit")
	defer srv.Close()

	c := New(Config{
		Endpoint:    srv.URL + "/v1",
		Model:       "test-model",
		APIKey:      "sekrit",
		QueryPrefix: "search_query: ",
	})
	vec, err := c.EmbedQuery("hello")
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	// First component encodes input length — prefix must have been applied.
	wantLen := float64(len("search_query: hello"))
	if vec[0] != wantLen {
		t.Fatalf("vec[0] = %v, want %v (prefix applied)", vec[0], wantLen)
	}
}

// captureServer records the last decoded request and returns 3-dim vectors.
func captureServer(t *testing.T, got *embedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		*got = req
		resp := embedResponse{}
		for i := range req.Input {
			resp.Data = append(resp.Data, struct {
				Index     int       `json:"index"`
				Embedding []float64 `json:"embedding"`
			}{Index: i, Embedding: []float64{1, 2, 3}})
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestEmbedOpenAI_CohereInputType(t *testing.T) {
	t.Parallel()
	var got embedRequest
	srv := captureServer(t, &got)
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "@bedrock/cohere.embed-english-v3"})
	if _, err := c.EmbedQuery("hi"); err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if got.InputType != "search_query" {
		t.Fatalf("query input_type = %q, want search_query", got.InputType)
	}
	if _, err := c.EmbedSessions(map[string]string{"s": "doc"}); err != nil {
		t.Fatalf("EmbedSessions: %v", err)
	}
	if got.InputType != "search_document" {
		t.Fatalf("document input_type = %q, want search_document", got.InputType)
	}
}

func TestEmbedOpenAI_NonCohereNoInputType(t *testing.T) {
	t.Parallel()
	var got embedRequest
	srv := captureServer(t, &got)
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "text-embedding-3-small"})
	if _, err := c.EmbedQuery("hi"); err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if got.InputType != "" {
		t.Fatalf("non-Cohere input_type = %q, want empty", got.InputType)
	}
}

func TestEmbedSessions_BatchesRequests(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	srv := fakeServer(t, &requests, "")
	defer srv.Close()

	sessions := make(map[string]string, BatchSize+10)
	for i := 0; i < BatchSize+10; i++ {
		sessions[fmt.Sprintf("s%03d", i)] = fmt.Sprintf("content %d", i)
	}

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "m"})
	got, err := c.EmbedSessions(sessions)
	if err != nil {
		t.Fatalf("EmbedSessions: %v", err)
	}
	if len(got) != len(sessions) {
		t.Fatalf("got %d vectors, want %d", len(got), len(sessions))
	}
	if n := requests.Load(); n != 2 {
		t.Fatalf("made %d requests for %d sessions, want 2 (batched)", n, len(sessions))
	}
	for id, vec := range got {
		if len(vec) != 3 {
			t.Fatalf("session %s: vector dim %d, want 3", id, len(vec))
		}
	}
}

func TestEmbed_TimeoutNeverStalls(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // stalled server
	}))
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "m", Timeout: 100 * time.Millisecond})
	start := time.Now()
	_, err := c.EmbedQuery("x")
	if err == nil {
		t.Fatal("expected timeout error from stalled server")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout took %v — must bail fast so the post-commit hook never stalls", elapsed)
	}
}

func TestEmbed_ServerErrorSurfaces(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "nope"})
	_, err := c.EmbedQuery("x")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v, want status surfaced", err)
	}
}

// fakeBedrock implements the Cohere-on-Bedrock invoke shape: it echoes the
// input_type into each vector's first component so tests can assert
// query/document routing, and returns embeddings under the Cohere key.
func fakeBedrock(t *testing.T, gotInputType *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/model/") || !strings.HasSuffix(r.URL.Path, "/invoke") {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		var req bedrockCohereRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if gotInputType != nil {
			*gotInputType = req.InputType
		}
		marker := 0.0
		if req.InputType == "search_query" {
			marker = 1.0
		}
		resp := bedrockCohereResponse{}
		for range req.Texts {
			resp.Embeddings = append(resp.Embeddings, []float64{marker, 2, 3})
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestEmbedBedrock_QueryVsDocumentInputType(t *testing.T) {
	t.Parallel()
	var inputType string
	srv := fakeBedrock(t, &inputType)
	defer srv.Close()

	c := New(Config{
		Provider: ProviderBedrock,
		Endpoint: srv.URL,
		Model:    "cohere.embed-english-v3",
		APIKey:   "bedrock-key",
	})

	qvec, err := c.EmbedQuery("how does retry backoff work")
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if inputType != "search_query" {
		t.Fatalf("query input_type = %q, want search_query", inputType)
	}
	if qvec[0] != 1.0 {
		t.Fatalf("query marker = %v, want 1", qvec[0])
	}

	got, err := c.EmbedSessions(map[string]string{"s1": "doc one", "s2": "doc two"})
	if err != nil {
		t.Fatalf("EmbedSessions: %v", err)
	}
	if inputType != "search_document" {
		t.Fatalf("session input_type = %q, want search_document", inputType)
	}
	if len(got) != 2 {
		t.Fatalf("got %d vectors, want 2", len(got))
	}
}

func TestEmbedBedrock_ErrorMessageSurfaces(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(bedrockCohereResponse{Message: "malformed input"})
	}))
	defer srv.Close()

	c := New(Config{Provider: ProviderBedrock, Endpoint: srv.URL, Model: "cohere.embed-english-v3"})
	_, err := c.EmbedQuery("x")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v, want status surfaced", err)
	}
}

func TestEmbed_CountMismatchRejected(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embedResponse{}) // zero vectors for N inputs
	}))
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL + "/v1", Model: "m"})
	if _, err := c.EmbedQuery("x"); err == nil {
		t.Fatal("expected error when server returns fewer vectors than inputs")
	}
}
