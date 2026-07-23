package graph

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// storeRoot returns a gitRoot whose StoreDir (<root>/.rekal) exists, so Append's
// O_CREATE has a directory to write into.
func storeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		t.Fatalf("mkdir .rekal: %v", err)
	}
	return root
}

func TestAppendDrainRoundTrip(t *testing.T) {
	t.Parallel()
	root := storeRoot(t)
	now := time.Now().UTC().Truncate(time.Second)
	in := []Edge{
		{TS: now, Kind: "recall", Query: "jwt expiry", Target: "S1"},
		{TS: now, Kind: "recall", Query: "jwt expiry", Target: "S2"},
		{TS: now, Kind: "drill", Target: "S1"},
	}
	if err := Append(root, in); err != nil {
		t.Fatalf("append: %v", err)
	}
	out, err := Drain(root)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("drained %d edges, want 3", len(out))
	}
	if out[0].Target != "S1" || out[0].Kind != "recall" || out[0].Query != "jwt expiry" {
		t.Fatalf("edge 0 = %+v", out[0])
	}
	if out[2].Kind != "drill" || out[2].Query != "" {
		t.Fatalf("edge 2 = %+v", out[2])
	}
	// Drain claims the spool: a second drain finds nothing.
	again, err := Drain(root)
	if err != nil {
		t.Fatalf("second drain: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second drain returned %d, want 0", len(again))
	}
}

func TestDrainMissingSpool(t *testing.T) {
	t.Parallel()
	root := storeRoot(t)
	out, err := Drain(root)
	if err != nil {
		t.Fatalf("drain on missing spool errored: %v", err)
	}
	if out != nil {
		t.Fatalf("drain on missing spool returned %v, want nil", out)
	}
}

func TestDrainToleratesPartialLine(t *testing.T) {
	t.Parallel()
	root := storeRoot(t)
	// A valid line, then a truncated one (an append caught mid-write).
	path := filepath.Join(root, ".rekal", spoolName)
	content := `{"ts":"2026-07-23T00:00:00Z","kind":"recall","query":"q","target":"S1"}` + "\n" +
		`{"ts":"2026-07-23T00:00:00Z","kind":"recall","query":"partial`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spool: %v", err)
	}
	out, err := Drain(root)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(out) != 1 || out[0].Target != "S1" {
		t.Fatalf("drain tolerating partial line = %+v, want 1 valid edge", out)
	}
}

func TestDrainSkipsTargetlessEdge(t *testing.T) {
	t.Parallel()
	root := storeRoot(t)
	path := filepath.Join(root, ".rekal", spoolName)
	content := `{"ts":"2026-07-23T00:00:00Z","kind":"recall","query":"q","target":""}` + "\n" +
		`{"ts":"2026-07-23T00:00:00Z","kind":"drill","target":"S9"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spool: %v", err)
	}
	out, err := Drain(root)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(out) != 1 || out[0].Target != "S9" {
		t.Fatalf("drain = %+v, want only the S9 edge", out)
	}
}

func TestAppendEmptyIsNoop(t *testing.T) {
	t.Parallel()
	root := storeRoot(t)
	if err := Append(root, nil); err != nil {
		t.Fatalf("append nil: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".rekal", spoolName)); !os.IsNotExist(err) {
		t.Fatalf("empty append created a spool file")
	}
}

// Not parallel: mutates process env via t.Setenv.
func TestAppendBenchGateSuppresses(t *testing.T) {
	root := storeRoot(t)
	t.Setenv("REKAL_BENCH", "1")
	if err := Append(root, []Edge{{TS: time.Now(), Kind: "recall", Target: "S1"}}); err != nil {
		t.Fatalf("append under bench: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".rekal", spoolName)); !os.IsNotExist(err) {
		t.Fatalf("bench-gated append still wrote a spool file")
	}
}
