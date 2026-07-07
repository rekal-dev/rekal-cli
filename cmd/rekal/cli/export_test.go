package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
)

// TestExportNewFrames_DoesNotMarkExportedBeforeCommit is the regression test
// for the export-ordering bug: transport.ExportNewFrames must return the wire-format
// bytes plus the exported checkpoint IDs WITHOUT marking those checkpoints
// exported in data.db. Only the caller, after transport.CommitWireFormat durably
// commits the bytes to the orphan branch, should mark them. If a crash or
// git failure happened between the old export-then-mark and an actual
// commit, a checkpoint could be permanently flagged exported without its
// content ever reaching the orphan branch — silently never syncing to
// teammates. This test proves the fix: mid-way through the export→commit→
// mark sequence, the checkpoint is still unexported and would be retried.
func TestExportNewFrames_DoesNotMarkExportedBeforeCommit(t *testing.T) {
	gitRoot := setupExportTestRepo(t)

	sessionID := ulid.Make().String()
	checkpointID := ulid.Make().String()
	now := time.Now().UTC().Format(time.RFC3339)

	seedCheckpoint(t, gitRoot, sessionID, checkpointID, now)

	// Sanity: checkpoint starts unexported.
	if got := countUnexported(t, gitRoot); got != 1 {
		t.Fatalf("before export: got %d unexported checkpoints, want 1", got)
	}

	body, dict, exportedIDs, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("transport.ExportNewFrames: %v", err)
	}
	if body == nil {
		t.Fatal("transport.ExportNewFrames returned nil body for a checkpoint that should have been exported")
	}
	if len(exportedIDs) != 1 || exportedIDs[0] != checkpointID {
		t.Fatalf("exportedIDs = %v, want [%s]", exportedIDs, checkpointID)
	}

	// The regression check: transport.ExportNewFrames must NOT have marked the
	// checkpoint exported yet — transport.CommitWireFormat hasn't run.
	if got := countUnexported(t, gitRoot); got != 1 {
		t.Fatalf("after transport.ExportNewFrames (before commit): got %d unexported checkpoints, want 1 (still unexported)", got)
	}

	// Now commit and mark, mirroring doPush's ordering.
	if _, err := transport.CommitWireFormat(gitRoot, body, dict); err != nil {
		t.Fatalf("transport.CommitWireFormat: %v", err)
	}
	if err := markCheckpointsExported(gitRoot, exportedIDs); err != nil {
		t.Fatalf("markCheckpointsExported: %v", err)
	}

	if got := countUnexported(t, gitRoot); got != 0 {
		t.Fatalf("after commit+mark: got %d unexported checkpoints, want 0", got)
	}
}

// TestExportNewFrames_CrashBetweenExportAndMarkIsRetried simulates a process
// dying after transport.ExportNewFrames but before transport.CommitWireFormat/markCheckpointsExported
// ever run, by simply never calling them, then re-running the whole export
// path fresh — the checkpoint must still be picked up (not silently lost).
func TestExportNewFrames_CrashBetweenExportAndMarkIsRetried(t *testing.T) {
	gitRoot := setupExportTestRepo(t)

	sessionID := ulid.Make().String()
	checkpointID := ulid.Make().String()
	now := time.Now().UTC().Format(time.RFC3339)

	seedCheckpoint(t, gitRoot, sessionID, checkpointID, now)

	// First "attempt": export runs, then the process is imagined to die
	// before commit/mark — simply don't call them.
	_, _, exportedIDs, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("transport.ExportNewFrames (first attempt): %v", err)
	}
	if len(exportedIDs) != 1 {
		t.Fatalf("first attempt exportedIDs = %v, want 1 entry", exportedIDs)
	}

	// Retry: a fresh export must still find the same checkpoint unexported.
	body2, dict2, exportedIDs2, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("transport.ExportNewFrames (retry): %v", err)
	}
	if len(exportedIDs2) != 1 || exportedIDs2[0] != checkpointID {
		t.Fatalf("retry exportedIDs = %v, want [%s] — checkpoint was lost after simulated crash", exportedIDs2, checkpointID)
	}

	if _, err := transport.CommitWireFormat(gitRoot, body2, dict2); err != nil {
		t.Fatalf("transport.CommitWireFormat: %v", err)
	}
	if err := markCheckpointsExported(gitRoot, exportedIDs2); err != nil {
		t.Fatalf("markCheckpointsExported: %v", err)
	}

	if got := countUnexported(t, gitRoot); got != 0 {
		t.Fatalf("after retry commit+mark: got %d unexported checkpoints, want 0", got)
	}
}

// TestExportNewFrames_MergedOnlyGate proves the merged-only sharing guarantee
// end to end (docs/design/merged-only-sharing.md): a checkpoint on an unmerged
// branch is held back by export — staying unexported — and is released
// automatically by the next export after its branch merges into main. This is
// the "self-releasing" property: no promote step, no on-merge hook.
func TestExportNewFrames_MergedOnlyGate(t *testing.T) {
	gitRoot := setupExportTestRepo(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Checkpoint A: on main (merged by definition).
	mergedSession := ulid.Make().String()
	mergedCP := ulid.Make().String()
	seedCheckpointAt(t, gitRoot, mergedSession, mergedCP, now, headSHA(t, gitRoot), "main")

	// Checkpoint B: on a feature branch main has never seen.
	runGit(t, gitRoot, "checkout", "-b", "feature")
	runGit(t, gitRoot, "commit", "--allow-empty", "-m", "wip on feature")
	featureSHA := headSHA(t, gitRoot)
	runGit(t, gitRoot, "checkout", "main")
	unmergedSession := ulid.Make().String()
	unmergedCP := ulid.Make().String()
	seedCheckpointAt(t, gitRoot, unmergedSession, unmergedCP, now, featureSHA, "feature")

	// First push: only the merged checkpoint may reach the wire.
	body, dict, exportedIDs, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("ExportNewFrames: %v", err)
	}
	if len(exportedIDs) != 1 || exportedIDs[0] != mergedCP {
		t.Fatalf("exportedIDs = %v, want only the merged checkpoint %s", exportedIDs, mergedCP)
	}
	if _, err := transport.CommitWireFormat(gitRoot, body, dict); err != nil {
		t.Fatalf("CommitWireFormat: %v", err)
	}
	if err := markCheckpointsExported(gitRoot, exportedIDs); err != nil {
		t.Fatalf("markCheckpointsExported: %v", err)
	}

	// The held-back checkpoint must still be unexported — not lost, not leaked.
	if got := countUnexported(t, gitRoot); got != 1 {
		t.Fatalf("after first push: %d unexported checkpoints, want 1 (the unmerged one held back)", got)
	}

	// A push with nothing merged exports nothing (held-back only ⇒ nil body).
	body2, _, exportedIDs2, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("ExportNewFrames (held-back only): %v", err)
	}
	if body2 != nil || len(exportedIDs2) != 0 {
		t.Fatalf("held-back-only export shipped something: ids=%v", exportedIDs2)
	}

	// Merge the feature branch. The next export must release the held-back
	// checkpoint with no other intervention.
	runGit(t, gitRoot, "merge", "--no-ff", "-m", "merge feature", "feature")

	body3, dict3, exportedIDs3, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		t.Fatalf("ExportNewFrames (after merge): %v", err)
	}
	if len(exportedIDs3) != 1 || exportedIDs3[0] != unmergedCP {
		t.Fatalf("after merge exportedIDs = %v, want [%s] — held-back checkpoint was not released", exportedIDs3, unmergedCP)
	}
	if _, err := transport.CommitWireFormat(gitRoot, body3, dict3); err != nil {
		t.Fatalf("CommitWireFormat (after merge): %v", err)
	}
	if err := markCheckpointsExported(gitRoot, exportedIDs3); err != nil {
		t.Fatalf("markCheckpointsExported (after merge): %v", err)
	}
	if got := countUnexported(t, gitRoot); got != 0 {
		t.Fatalf("after merge push: %d unexported checkpoints, want 0", got)
	}
}

// seedCheckpoint inserts a minimal session+turn+checkpoint fixture, using its
// own short-lived data.db connection (each helper/production function in
// this path opens and closes its own connection — sharing one long-lived
// handle across transport.ExportNewFrames/markCheckpointsExported calls, which do the
// same, causes stale reads against a DuckDB file opened by multiple
// concurrent *sql.DB handles in one process).
func seedCheckpoint(t *testing.T, gitRoot, sessionID, checkpointID, ts string) {
	t.Helper()
	// The checkpoint's git_sha must be a real commit on the default branch so
	// the merged-only export gate shares it (see ExportNewFrames / filterMerged
	// in transport). setupExportTestRepo commits on and renames HEAD to main.
	seedCheckpointAt(t, gitRoot, sessionID, checkpointID, ts, headSHA(t, gitRoot), "main")
}

// seedCheckpointAt is seedCheckpoint pinned to a specific commit and branch —
// used by the merged-only gate test to seed checkpoints on unmerged branches.
func seedCheckpointAt(t *testing.T, gitRoot, sessionID, checkpointID, ts, gitSHA, branch string) {
	t.Helper()

	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	// Content hash must be unique per session (dedup is by hash).
	if err := db.InsertSession(dataDB, sessionID, "", "hash-"+sessionID, "human", "", "dev@example.com", branch, ts, "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	if err := db.InsertTurn(dataDB, ulid.Make().String(), sessionID, 0, "human", "fix the bug", ts); err != nil {
		t.Fatalf("InsertTurn: %v", err)
	}
	if err := db.InsertCheckpoint(dataDB, checkpointID, gitSHA, branch, "dev@example.com", ts, "human", ""); err != nil {
		t.Fatalf("InsertCheckpoint: %v", err)
	}
	if err := db.InsertCheckpointSession(dataDB, checkpointID, sessionID); err != nil {
		t.Fatalf("InsertCheckpointSession: %v", err)
	}
}

// countUnexported opens a fresh data.db connection to count unexported
// checkpoints — see seedCheckpoint's doc comment on why this must not reuse
// a connection held open across transport.ExportNewFrames/markCheckpointsExported.
func countUnexported(t *testing.T, gitRoot string) int {
	t.Helper()

	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	unexported, err := db.QueryUnexportedCheckpoints(dataDB)
	if err != nil {
		t.Fatalf("QueryUnexportedCheckpoints: %v", err)
	}
	return len(unexported)
}

// setupExportTestRepo creates a temp git repo with .rekal/data.db initialized
// and a local orphan branch ready for export/commit, and chdirs the test
// process into it (gitx.ConfigValue and gitx.RekalBranchName resolve git config
// relative to cwd, not gitRoot). Not safe to run in parallel with other
// tests that chdir.
func setupExportTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "dev@example.com")
	runGit(t, dir, "config", "user.name", "Dev")
	runGit(t, dir, "commit", "--allow-empty", "-m", "initial")
	// Normalize the default branch name so gitx.DefaultBranch resolves
	// deterministically regardless of the host's init.defaultBranch.
	runGit(t, dir, "branch", "-M", "main")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	rekalDir := RekalDir(dir)
	if err := os.MkdirAll(rekalDir, 0o755); err != nil {
		t.Fatalf("MkdirAll .rekal: %v", err)
	}

	dataDB, err := db.OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := db.InitDataSchema(dataDB); err != nil {
		dataDB.Close()
		t.Fatalf("InitDataSchema: %v", err)
	}
	dataDB.Close()

	if err := transport.EnsureOrphanBranch(dir); err != nil {
		t.Fatalf("transport.EnsureOrphanBranch: %v", err)
	}

	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// headSHA returns the current HEAD commit SHA of the repo at dir.
func headSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}
