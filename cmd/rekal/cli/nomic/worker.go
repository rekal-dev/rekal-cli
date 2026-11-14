package nomic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// socketUnavailable selects pipe IPC only when the host denies Unix sockets.
func socketUnavailable(err error) bool {
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EPROTONOSUPPORT)
}

// workerPipe isolates native inference in a child on socket-restricted hosts.
// A worker is reused for its client's batch, then terminated and reaped.
type workerPipe struct {
	reader io.ReadCloser
	writer io.WriteCloser
	cmd    *exec.Cmd
	once   sync.Once
}

func (p *workerPipe) Read(b []byte) (int, error)  { return p.reader.Read(b) }
func (p *workerPipe) Write(b []byte) (int, error) { return p.writer.Write(b) }
func (p *workerPipe) Close() error {
	p.once.Do(func() {
		_ = p.writer.Close()
		_ = p.reader.Close()
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
	})
	return nil
}

func newPipeClient(gitRoot string) (*daemonClient, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(exe, ".test") || strings.Contains(exe, "/_test/") {
		return nil, fmt.Errorf("nomic: cannot launch embedding worker from a test binary")
	}
	cmd := exec.Command(exe, "_nomic-daemon", "--stdio")
	if _, err := nomicDir(gitRoot); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(daemonLogPath(gitRoot), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	cmd.Stderr = f
	return startPipeClient(cmd, 30*time.Second)
}

func startPipeClient(cmd *exec.Cmd, timeout time.Duration) (*daemonClient, error) {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	writer, err := cmd.StdinPipe()
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, err
	}
	dc := &daemonClient{conn: &workerPipe{reader: reader, writer: writer, cmd: cmd}}
	ready := make(chan error, 1)
	go func() { ready <- dc.ping() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err = <-ready:
		if err == nil {
			return dc, nil
		}
	case <-timer.C:
		err = fmt.Errorf("startup exceeded %s", timeout)
	}
	dc.Close()
	return nil, fmt.Errorf("nomic: pipe worker unavailable: %w", err)
}

// RunStdioWorker serves framed requests over inherited pipes; EOF ends it.
func RunStdioWorker(reader io.Reader, writer io.Writer) error {
	SetVerbose(true)
	embedder, err := NewEmbedder()
	if err != nil {
		return fmt.Errorf("nomic worker: load model: %w", err)
	}
	defer embedder.Close()
	var mu sync.Mutex
	for {
		var req daemonRequest
		if err := readMsg(reader, &req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := writeMsg(writer, handleRequest(req, embedder, &mu)); err != nil {
			return err
		}
	}
}
