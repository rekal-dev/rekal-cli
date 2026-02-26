//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestSync_Team_NoRemote(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Sync with no remote configured should succeed gracefully.
	// It will checkpoint (no sessions), skip push (no remote), and rebuild index from local data.
	_, stderr, err := env.RunCLI("sync")
	if err != nil {
		t.Fatalf("sync should succeed: %v", err)
	}
	if !strings.Contains(stderr, "synced") {
		t.Errorf("expected sync summary, got: %q", stderr)
	}
	if !strings.Contains(stderr, "0 local sessions") {
		t.Errorf("expected 0 local sessions in summary, got: %q", stderr)
	}
}

func TestSync_Self_NoRemote(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Sync --self with no remote should fail (that's the whole point of --self).
	_, _, err := env.RunCLI("sync", "--self")
	if err == nil {
		t.Fatal("sync --self with no remote should fail")
	}
}

func TestSync_Team_RebuildIndex(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Run sync — it should rebuild the index even with no data.
	_, stderr, err := env.RunCLI("sync")
	if err != nil {
		t.Fatalf("sync should succeed: %v", err)
	}
	if !strings.Contains(stderr, "indexing local data") {
		t.Errorf("expected index rebuild message, got: %q", stderr)
	}
}
