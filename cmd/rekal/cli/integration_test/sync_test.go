//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
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

// TestSync_Team_FailedRebuildLeavesNoTempFile is the `rekal sync` equivalent
// of TestIndex_FailedRebuildLeavesLiveIndexIntact (recall_test.go):
// runSyncTeam has its own separate rebuild-into-a-temp-file implementation
// (it folds remote-branch import into the same pass rather than calling
// runIndex), so it needs its own regression coverage for the same
// atomicity property — a failed rebuild must not leave a stray
// `.rebuilding` temp file behind.
func TestSync_Team_FailedRebuildLeavesNoTempFile(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	if _, stderr, err := env.RunCLI("sync"); err != nil {
		t.Fatalf("initial sync should succeed: %v\nstderr: %s", err, stderr)
	}

	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	if _, err := dataDB.Exec("DROP TABLE turns"); err != nil {
		t.Fatalf("drop turns table: %v", err)
	}
	dataDB.Close()

	if _, stderr, err := env.RunCLI("sync"); err == nil {
		t.Fatalf("expected sync's rebuild to fail after corrupting data.db, but it succeeded\nstderr: %s", stderr)
	}

	tmpPath := db.IndexPath(env.RepoDir) + ".rebuilding"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Errorf("expected no leftover temp file at %s after a failed sync rebuild", tmpPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error stat-ing temp file: %v", err)
	}
}
