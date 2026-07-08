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
