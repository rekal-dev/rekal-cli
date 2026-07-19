//go:build (darwin && arm64) || (linux && amd64) || (linux && arm64)

package nomic

import (
	"errors"
	"fmt"
	"time"
)

// Client provides nomic embeddings through the daemon. The model is loaded
// ONLY in the daemon process, never in-process: a native model-load crash
// (a bad model, an incompatible llama build, an OOM) then kills the disposable
// daemon, not the caller. recall and embed degrade to keyword/LSA instead of
// dying — the memory tool must never fail harder than the fallback it guards.
// (The old in-process fallback ran the crashable CGO in the caller, so a
// SIGSEGV during model load took the whole `rekal recall` down with exit 2.)
type Client struct {
	daemon *daemonClient
}

// ErrDaemonNotReady means no serving daemon is up yet; one has been spawned in
// the background. Callers that can't wait (recall) treat this as "skip the
// semantic layer this call" and degrade; the daemon warms for next time.
var ErrDaemonNotReady = errors.New("nomic: embedding daemon warming up")

const (
	daemonReadyTimeout = 30 * time.Second
	daemonReadyPoll    = 400 * time.Millisecond
)

// NewClient returns a daemon-backed client. If a daemon is already serving it
// connects immediately. Otherwise it spawns one (model load happens there, off
// the caller's process) and then, when wait is true, blocks until the daemon
// serves or a bounded timeout elapses; when wait is false it returns
// ErrDaemonNotReady at once so an interactive caller degrades now and picks up
// the warm daemon on the next call.
func NewClient(gitRoot string, wait bool) (*Client, error) {
	if !Supported() {
		return nil, ErrNotSupported
	}
	if dc, err := connectDaemon(gitRoot); err == nil {
		return &Client{daemon: dc}, nil
	}

	spawnDaemon(gitRoot)
	if !wait {
		return nil, ErrDaemonNotReady
	}

	// Batch callers (embed/index/checkpoint) can wait for the model to load in
	// the daemon. If it never comes up (crash / unsupported build), return an
	// error — the caller treats missing vectors as non-fatal and warns.
	deadline := time.Now().Add(daemonReadyTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(daemonReadyPoll)
		if dc, err := connectDaemon(gitRoot); err == nil {
			return &Client{daemon: dc}, nil
		}
	}
	return nil, fmt.Errorf("nomic: daemon did not become ready within %s", daemonReadyTimeout)
}

// EmbedQuery embeds text with the "search_query: " prefix.
func (c *Client) EmbedQuery(text string) ([]float64, error) {
	return c.daemon.EmbedQuery(text)
}

// EmbedDocument embeds text with the "search_document: " prefix.
func (c *Client) EmbedDocument(text string) ([]float64, error) {
	return c.daemon.EmbedDocument(text)
}

// EmbedSessions embeds multiple sessions in batch.
func (c *Client) EmbedSessions(sessions map[string]string) (map[string][]float64, error) {
	return c.daemon.EmbedSessions(sessions)
}

// Close releases the daemon connection (the daemon itself lives on for reuse).
func (c *Client) Close() {
	if c.daemon != nil {
		c.daemon.Close()
	}
}
