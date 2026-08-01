package nomic

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSocketPath_ShortRootStaysInTheStore pins that the common case is
// unchanged: the socket belongs next to the lock, in the store.
func TestSocketPath_ShortRootStaysInTheStore(t *testing.T) {
	t.Parallel()

	root := "/home/u/proj"
	want := filepath.Join(root, ".rekal", "nomic", "daemon.sock")
	if got := socketPath(root); got != want {
		t.Errorf("socketPath(%q) = %q, want %q", root, got, want)
	}
}

// TestSocketPath_LongRootBindsAnyway is the bug this guards.
//
// sockaddr_un bounds the whole absolute path, so a repo nested deeply enough —
// a CI workspace, a container mount, an iCloud path — pushes the in-store
// socket past the limit and bind fails with EINVAL. The daemon loaded the
// model and then died on listen, every spawn, forever, while the caller saw
// only "daemon did not become ready" and recall told the agent to retry with
// backoff. It could never succeed.
//
// The assertion is the real one: bind the path, on this machine's kernel.
func TestSocketPath_LongRootBindsAnyway(t *testing.T) {
	// No t.Parallel: this test sets XDG_RUNTIME_DIR.

	// A root long enough that <root>/.rekal/nomic/daemon.sock exceeds the
	// limit, which is what a deep checkout actually looks like.
	root := "/" + strings.Repeat("nested-workspace-directory/", 5) + "repo"
	inStore := filepath.Join(root, ".rekal", "nomic", "daemon.sock")
	if len(inStore) <= maxUnixSocketPath {
		t.Fatalf("fixture root is too short to exercise the limit: %d bytes", len(inStore))
	}

	got := socketPath(root)
	if len(got) > maxUnixSocketPath {
		t.Fatalf("socketPath returned %d bytes (%q), over the %d-byte limit", len(got), got, maxUnixSocketPath)
	}

	// Redirect the fallback into the test's own directory so the assertion does
	// not depend on, or litter, a shared runtime dir.
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	got = socketPath(root)
	ln, err := net.Listen("unix", got)
	if err != nil {
		t.Fatalf("cannot bind %q (%d bytes): %v", got, len(got), err)
	}
	defer ln.Close() //nolint:errcheck

	// Both sides derive the rendezvous from the store root alone, or the client
	// would dial a socket no daemon is listening on.
	if again := socketPath(root); again != got {
		t.Errorf("socketPath is not deterministic: %q then %q", got, again)
	}
	// And two stores must not share one.
	if other := socketPath(root + "-two"); other == got {
		t.Error("two different stores resolved to the same socket — one daemon would serve both")
	}
}

// TestSocketPath_FallbackDirIsPrivate pins the permissions on the shared-dir
// fallback: the temp dir is world-writable, the socket's parent must not be.
func TestSocketPath_FallbackDirIsPrivate(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	root := "/" + strings.Repeat("nested-workspace-directory/", 5) + "repo"
	got := socketPath(root)

	fi, err := os.Stat(filepath.Dir(got))
	if err != nil {
		t.Fatalf("stat fallback dir: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("fallback socket dir is %o, want 0700 — it may sit in a shared temp dir", perm)
	}
}
