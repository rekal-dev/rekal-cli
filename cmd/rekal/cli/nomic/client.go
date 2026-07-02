//go:build (darwin && arm64) || (linux && amd64) || (linux && arm64)

package nomic

// Client provides nomic embeddings, transparently using the daemon when
// available or falling back to in-process embedding.
type Client struct {
	daemon   *daemonClient
	embedder *Embedder
}

// NewClient creates a Client that tries the daemon first for fast embedding.
// If no daemon is running, it falls back to loading the model in-process and
// spawns a daemon in the background for future invocations.
// gitRoot is the git repository root (used to locate .rekal/nomic/).
func NewClient(gitRoot string) (*Client, error) {
	if !Supported() {
		return nil, ErrNotSupported
	}

	// Try connecting to a running daemon.
	dc, err := connectDaemon(gitRoot)
	if err == nil {
		return &Client{daemon: dc}, nil
	}

	// No daemon running — fall back to in-process and spawn one for next time.
	embedder, err := NewEmbedder()
	if err != nil {
		return nil, err
	}
	spawnDaemon(gitRoot)
	return &Client{embedder: embedder}, nil
}

// EmbedQuery embeds text with the "search_query: " prefix.
func (c *Client) EmbedQuery(text string) ([]float64, error) {
	if c.daemon != nil {
		return c.daemon.EmbedQuery(text)
	}
	return c.embedder.EmbedQuery(text)
}

// EmbedDocument embeds text with the "search_document: " prefix.
func (c *Client) EmbedDocument(text string) ([]float64, error) {
	if c.daemon != nil {
		return c.daemon.EmbedDocument(text)
	}
	return c.embedder.EmbedDocument(text)
}

// EmbedSessions embeds multiple sessions in batch.
func (c *Client) EmbedSessions(sessions map[string]string) (map[string][]float64, error) {
	if c.daemon != nil {
		return c.daemon.EmbedSessions(sessions)
	}
	return c.embedder.EmbedSessions(sessions)
}

// Close releases resources.
func (c *Client) Close() {
	if c.daemon != nil {
		c.daemon.Close()
	}
	if c.embedder != nil {
		c.embedder.Close()
	}
}
