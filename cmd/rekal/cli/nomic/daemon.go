package nomic

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	idleTimeout = 5 * time.Minute
	dialTimeout = 2 * time.Second
)

// daemonRequest is the JSON wire format for client→daemon messages.
type daemonRequest struct {
	Op       string            `json:"op"`                 // ping, embed_query, embed_document, embed_sessions
	Text     string            `json:"text,omitempty"`     // for embed_query, embed_document
	Sessions map[string]string `json:"sessions,omitempty"` // for embed_sessions
}

// daemonResponse is the JSON wire format for daemon→client messages.
type daemonResponse struct {
	OK      bool                 `json:"ok"`
	Error   string               `json:"error,omitempty"`
	Vector  []float64            `json:"vector,omitempty"`  // for embed_query, embed_document
	Vectors map[string][]float64 `json:"vectors,omitempty"` // for embed_sessions
}

// nomicDir returns .rekal/nomic/ under the given git root, creating it if needed.
func nomicDir(gitRoot string) (string, error) {
	dir := filepath.Join(gitRoot, ".rekal", "nomic")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("nomic: create dir: %w", err)
	}
	return dir, nil
}

// maxUnixSocketPath is the portable ceiling for a Unix socket path. sockaddr_un
// holds 108 bytes on Linux and 104 on macOS/BSD, each including the terminating
// NUL — 103 is the largest length that binds everywhere.
const maxUnixSocketPath = 103

// socketPath is where the daemon listens for a store.
//
// It normally lives beside the lock, in the store. But a Unix socket path is
// bounded by sockaddr_un, and the bound counts the *whole* absolute path — so a
// repo nested deeply enough (a CI workspace, a container mount, an iCloud
// path) pushes it past the limit and bind fails with EINVAL, "invalid
// argument". The daemon would load the model, then die on listen, on every
// spawn forever. Nothing surfaced: the caller only ever saw "daemon did not
// become ready", which recall reports as `SEMANTIC warming — retry with
// backoff` — advice that can never succeed on such a repo.
//
// Past the ceiling the socket moves to a short runtime directory under a name
// derived from the store root, so daemon and client still agree on one
// rendezvous per store. The fallback path itself is length-checked: macOS
// TempDir() and a deep XDG_RUNTIME_DIR can still exceed the bound, so those
// land under /tmp/rekal-nomic-<hash>/ instead. The single-flight guarantee is
// unaffected: it comes from the flock on .rekal/nomic/daemon.lock, which is a
// regular file and has no length limit.
func socketPath(gitRoot string) string {
	p := filepath.Join(gitRoot, ".rekal", "nomic", "daemon.sock")
	if len(p) <= maxUnixSocketPath {
		return p
	}
	sum := sha256.Sum256([]byte(gitRoot))
	hex := fmt.Sprintf("%x", sum[:8])
	candidates := []string{
		filepath.Join(runtimeDir(), "rekal-nomic", hex+".sock"),
		filepath.Join("/tmp", "rekal-nomic-"+hex, "d.sock"),
	}
	for _, c := range candidates {
		if len(c) > maxUnixSocketPath {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(c), 0o700) //nolint:errcheck
		return c
	}
	// Unreachable: /tmp/rekal-nomic-<16 hex>/d.sock is ~36 bytes.
	p = candidates[len(candidates)-1]
	_ = os.MkdirAll(filepath.Dir(p), 0o700) //nolint:errcheck
	return p
}

// runtimeDir prefers the per-user runtime directory when the platform provides
// one; it is already private, where the temp dir is shared.
func runtimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	return os.TempDir()
}

func pidPath(gitRoot string) string {
	return filepath.Join(gitRoot, ".rekal", "nomic", "daemon.pid")
}

// daemonLogPath is where a spawned daemon's stderr lands, so a start-up failure
// leaves a trace instead of presenting as a daemon that never warms.
func daemonLogPath(gitRoot string) string {
	return filepath.Join(gitRoot, ".rekal", "nomic", "daemon.log")
}

// writeMsg writes a length-prefixed JSON message to a connection.
func writeMsg(conn net.Conn, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := conn.Write(hdr[:]); err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}

// readMsg reads a length-prefixed JSON message from a connection.
func readMsg(conn net.Conn, v interface{}) error {
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(hdr[:])
	if size > 64*1024*1024 { // 64 MB sanity limit
		return fmt.Errorf("nomic daemon: message too large (%d bytes)", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}

// RunDaemon runs the nomic embedding daemon. It takes a single-flight lock,
// loads the model, THEN opens the socket, and exits after idleTimeout of
// inactivity. The ordering matters: a connectable socket therefore always means
// a ready daemon, and the lock means exactly one daemon serves a store — so
// concurrent recall/embed calls can't spawn a swarm of daemons racing to
// re-listen and reload the model (the crash that made recall flaky).
func RunDaemon(gitRoot string) error {
	dir, err := nomicDir(gitRoot)
	if err != nil {
		return err
	}

	// Single-flight: if another daemon already holds the lock, exit cleanly —
	// the running one owns the socket. Held for this process's whole lifetime.
	lock, ok, err := lockFile(filepath.Join(dir, "daemon.lock"), false)
	if err != nil {
		return err
	}
	if !ok {
		return nil // another daemon is serving this store
	}
	defer unlockFile(lock)

	sock := socketPath(gitRoot)
	pid := pidPath(gitRoot)

	// Load the model BEFORE opening the socket. Clients probe for the socket to
	// decide readiness; if it appeared first they could connect mid-load and
	// hang or race a crash. Load under the lock so no other daemon loads too.
	embedder, err := NewEmbedder()
	if err != nil {
		return fmt.Errorf("nomic daemon: load model: %w", err)
	}
	defer embedder.Close()

	// Now that the model is ready, publish the socket (clearing any stale one
	// left by a crashed predecessor).
	os.Remove(sock) //nolint:errcheck
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("nomic daemon: listen: %w", err)
	}
	defer ln.Close()      //nolint:errcheck
	defer os.Remove(sock) //nolint:errcheck
	defer os.Remove(pid)  //nolint:errcheck

	if err := os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return fmt.Errorf("nomic daemon: write pid: %w", err)
	}

	var mu sync.Mutex
	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	// Accept loop in goroutine; main goroutine watches idle timer.
	connCh := make(chan net.Conn)
	errCh := make(chan error, 1)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			connCh <- c
		}
	}()

	for {
		select {
		case conn := <-connCh:
			idle.Reset(idleTimeout)
			go handleConn(conn, embedder, &mu, idle)

		case <-idle.C:
			return nil // clean shutdown

		case err := <-errCh:
			if strings.Contains(err.Error(), "use of closed") {
				return nil
			}
			return err
		}
	}
}

func handleConn(conn net.Conn, embedder *Embedder, mu *sync.Mutex, idle *time.Timer) {
	defer conn.Close() //nolint:errcheck

	// Handle multiple requests on the same connection.
	for {
		var req daemonRequest
		if err := readMsg(conn, &req); err != nil {
			return // connection closed or read error
		}

		idle.Reset(idleTimeout)

		resp := handleRequest(req, embedder, mu)
		if err := writeMsg(conn, resp); err != nil {
			return
		}
	}
}

func handleRequest(req daemonRequest, embedder *Embedder, mu *sync.Mutex) daemonResponse {
	switch req.Op {
	case "ping":
		return daemonResponse{OK: true}

	case "embed_query":
		mu.Lock()
		vec, err := embedder.EmbedQuery(req.Text)
		mu.Unlock()
		if err != nil {
			return daemonResponse{Error: err.Error()}
		}
		return daemonResponse{OK: true, Vector: vec}

	case "embed_document":
		mu.Lock()
		vec, err := embedder.EmbedDocument(req.Text)
		mu.Unlock()
		if err != nil {
			return daemonResponse{Error: err.Error()}
		}
		return daemonResponse{OK: true, Vector: vec}

	case "embed_sessions":
		mu.Lock()
		vecs, err := embedder.EmbedSessions(req.Sessions)
		mu.Unlock()
		if err != nil {
			return daemonResponse{Error: err.Error()}
		}
		return daemonResponse{OK: true, Vectors: vecs}

	default:
		return daemonResponse{Error: fmt.Sprintf("unknown op: %s", req.Op)}
	}
}

// daemonClient wraps a Unix socket connection to the daemon.
type daemonClient struct {
	conn net.Conn
}

func (c *daemonClient) ping() error {
	if err := writeMsg(c.conn, daemonRequest{Op: "ping"}); err != nil {
		return err
	}
	var resp daemonResponse
	if err := readMsg(c.conn, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("ping failed: %s", resp.Error)
	}
	return nil
}

// EmbedQuery sends a query embedding request to the daemon.
func (c *daemonClient) EmbedQuery(text string) ([]float64, error) {
	if err := writeMsg(c.conn, daemonRequest{Op: "embed_query", Text: text}); err != nil {
		return nil, err
	}
	var resp daemonResponse
	if err := readMsg(c.conn, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("nomic daemon: %s", resp.Error)
	}
	return resp.Vector, nil
}

// EmbedDocument sends a document embedding request to the daemon.
func (c *daemonClient) EmbedDocument(text string) ([]float64, error) {
	if err := writeMsg(c.conn, daemonRequest{Op: "embed_document", Text: text}); err != nil {
		return nil, err
	}
	var resp daemonResponse
	if err := readMsg(c.conn, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("nomic daemon: %s", resp.Error)
	}
	return resp.Vector, nil
}

// EmbedSessions sends a batch session embedding request to the daemon.
func (c *daemonClient) EmbedSessions(sessions map[string]string) (map[string][]float64, error) {
	if err := writeMsg(c.conn, daemonRequest{Op: "embed_sessions", Sessions: sessions}); err != nil {
		return nil, err
	}
	var resp daemonResponse
	if err := readMsg(c.conn, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("nomic daemon: %s", resp.Error)
	}
	return resp.Vectors, nil
}

func (c *daemonClient) Close() {
	c.conn.Close() //nolint:errcheck
}

// connectDaemon tries to connect to a running daemon.
// Returns a connected daemonClient, or an error if no daemon is reachable.
// Does NOT spawn a new daemon — use spawnDaemon for that.
func connectDaemon(gitRoot string) (*daemonClient, error) {
	sock := socketPath(gitRoot)

	conn, err := net.DialTimeout("unix", sock, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("nomic: no daemon running")
	}
	dc := &daemonClient{conn: conn}
	if err := dc.ping(); err != nil {
		dc.Close()
		return nil, fmt.Errorf("nomic: daemon not responding: %w", err)
	}
	return dc, nil
}

// spawnDaemon launches a daemon process in the background.
// It does not wait for the daemon to become ready — callers should
// fall back to in-process embedding and benefit from the daemon on
// subsequent invocations.
// spawnCooldown rate-limits daemon spawns. With the daemon-only client, a
// caller that can't connect spawns a daemon on every miss; if the model can't
// load (bad model / incompatible build) the daemon crashes on startup and the
// next call would spawn another. The cooldown stops that from becoming a
// process storm on a broken environment — one spawn attempt per window.
const spawnCooldown = 15 * time.Second

func spawnDaemon(gitRoot string) {
	sock := socketPath(gitRoot)

	// Rate-limit: skip if we spawned within the cooldown (a daemon is either
	// warming up or repeatedly crashing — either way, don't pile on).
	stamp := filepath.Join(filepath.Dir(sock), "spawn.stamp")
	if fi, err := os.Stat(stamp); err == nil && time.Since(fi.ModTime()) < spawnCooldown {
		return
	}
	_ = os.MkdirAll(filepath.Dir(stamp), 0o755)                                      //nolint:errcheck
	_ = os.WriteFile(stamp, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o644) //nolint:errcheck

	// Note: stale socket/pid cleanup is the daemon's job now — it happens after
	// the daemon takes the single-flight lock, so we never delete a socket a
	// live (or loading) daemon owns.

	exe, err := os.Executable()
	if err != nil {
		return
	}

	// Don't spawn from test binaries — they can't serve the daemon command.
	if strings.HasSuffix(exe, ".test") || strings.Contains(exe, "/_test/") {
		return
	}

	cmd := exec.Command(exe, "_nomic-daemon", "--git-root", gitRoot)
	cmd.Stdout = nil
	// Keep the child's stderr. A daemon that can never start is
	// indistinguishable from one still warming up: the caller times out, recall
	// says `SEMANTIC warming — retry with backoff`, and the agent retries
	// forever against a machine where the model will never load. Discarding the
	// one place the reason exists made two real failures — an illegal
	// instruction from a mismatched CPU build, and a socket path over the
	// sockaddr_un limit — invisible on a working install. Truncated per spawn,
	// so it holds the latest attempt and cannot grow.
	if f, err := os.OpenFile(daemonLogPath(gitRoot), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		cmd.Stderr = f
		defer f.Close() //nolint:errcheck
	}
	cmd.Stdin = nil
	setSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return
	}
	// Fully detach — don't wait on the child.
	go cmd.Wait() //nolint:errcheck
}

// NewDaemonCmd returns the hidden _nomic-daemon cobra command.
func NewDaemonCmd() *cobra.Command {
	var gitRoot string

	cmd := &cobra.Command{
		Use:    "_nomic-daemon",
		Hidden: true,
		Short:  "Run the nomic embedding daemon (internal)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if gitRoot == "" {
				return fmt.Errorf("--git-root is required")
			}
			return RunDaemon(gitRoot)
		},
	}
	cmd.Flags().StringVar(&gitRoot, "git-root", "", "Git repository root")
	return cmd
}
