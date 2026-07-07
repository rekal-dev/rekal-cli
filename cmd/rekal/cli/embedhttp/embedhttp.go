// Package embedhttp is the HTTP embedding backend: an OpenAI-compatible
// /embeddings client (vLLM, Ollama, LM Studio, TEI, llamafile all speak it)
// as an alternative to the embedded nomic model. Pointed at a localhost
// server, "the data never leaves the machine" still holds — pointed anywhere
// else, session text leaves the machine, which is the caller's deliberate,
// documented choice (see docs/spec/command/recall.md).
//
// Two constraints shape this client:
//   - It runs inside the post-commit hook (incremental index update), so it
//     must never stall a commit: every request carries a hard timeout and any
//     failure is returned to the caller, which treats embedding as non-fatal.
//   - It must be fast against remote servers: sessions are embedded in
//     batches (BatchSize inputs per request), not one call per session.
package embedhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultTimeout bounds each HTTP request. Deliberately small: the
	// post-commit hook path embeds a handful of new sessions, and a stalled
	// server must fail the embedding step (non-fatal) rather than the commit.
	DefaultTimeout = 10 * time.Second

	// BatchSize is how many inputs ride one /embeddings request. Large enough
	// to amortize round-trips against a remote GPU server, small enough to
	// stay inside typical request-size limits.
	BatchSize = 64
)

// Config configures the HTTP embedding client. Values come from the
// "embedding" section of .rekal/config.json (resolved by the cli package,
// including env expansion — this package never reads the environment).
type Config struct {
	// Endpoint is the API base URL up to and including the version prefix,
	// e.g. "http://127.0.0.1:8000/v1". "/embeddings" is appended.
	Endpoint string
	// Model is the model name sent with each request and the key vectors are
	// stored under in session_embeddings.
	Model string
	// APIKey, when non-empty, is sent as a Bearer token.
	APIKey string
	// Timeout bounds each request; zero means DefaultTimeout.
	Timeout time.Duration
	// QueryPrefix/DocumentPrefix are prepended to query/document text.
	// Nomic-family models require "search_query: "/"search_document: ".
	QueryPrefix    string
	DocumentPrefix string
}

// Client is an OpenAI-compatible embeddings client. It mirrors the embedded
// nomic client's surface (EmbedQuery/EmbedSessions/Close/ModelName) so the
// index and recall paths can treat the two backends interchangeably.
type Client struct {
	cfg  Config
	http *http.Client
}

// New returns a client for the given config. It performs no I/O — a dead
// endpoint surfaces on the first Embed call, bounded by the timeout.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// ModelName returns the model vectors are stored under.
func (c *Client) ModelName() string { return c.cfg.Model }

// Close exists for symmetry with the embedded backend; nothing to release.
func (c *Client) Close() {}

// EmbedQuery embeds a recall query.
func (c *Client) EmbedQuery(text string) ([]float64, error) {
	vecs, err := c.embed([]string{c.cfg.QueryPrefix + text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedSessions embeds session documents, batched BatchSize per request.
func (c *Client) EmbedSessions(sessions map[string]string) (map[string][]float64, error) {
	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}

	out := make(map[string][]float64, len(sessions))
	for start := 0; start < len(ids); start += BatchSize {
		end := min(start+BatchSize, len(ids))
		batch := ids[start:end]

		inputs := make([]string, len(batch))
		for i, id := range batch {
			inputs[i] = c.cfg.DocumentPrefix + sessions[id]
		}
		vecs, err := c.embed(inputs)
		if err != nil {
			return nil, err
		}
		for i, id := range batch {
			out[id] = vecs[i]
		}
	}
	return out, nil
}

// embedRequest/embedResponse are the OpenAI-compatible wire shapes.
type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// embed sends one /embeddings request and returns vectors in input order.
func (c *Client) embed(inputs []string) ([][]float64, error) {
	body, err := json.Marshal(embedRequest{Model: c.cfg.Model, Input: inputs})
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/embeddings"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding endpoint: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding endpoint returned %s: %s", resp.Status, truncate(string(respBody), 200))
	}

	var parsed embedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse embedding response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("embedding endpoint error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding endpoint returned %d vectors for %d inputs", len(parsed.Data), len(inputs))
	}

	// Order by the response's index field — the API guarantees it matches
	// input positions, but some servers reorder data entries.
	out := make([][]float64, len(inputs))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(inputs) {
			return nil, fmt.Errorf("embedding endpoint returned out-of-range index %d", d.Index)
		}
		out[d.Index] = d.Embedding
	}
	for i, v := range out {
		if len(v) == 0 {
			return nil, fmt.Errorf("embedding endpoint returned no vector for input %d", i)
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
