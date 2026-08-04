package transport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestMergeGateCache_ReusedAndInvalidated covers the memoization on the merged-
// only gate.
//
// A checkpoint held back because its branch was abandoned is re-litigated on
// every push, and the squash probe costs a commit-tree plus a git cherry each
// time to reach the same answer. Caching it is only safe under two conditions,
// which this pins: the verdict is keyed to the mainline tip, so work that lands
// later is re-evaluated rather than held on a stale "no"; and the cache is an
// accelerator, never a source of truth — it may not let anything through that
// the gate itself would refuse.
func TestMergeGateCache_ReusedAndInvalidated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "base")

	// Work on a branch that is never merged.
	gitIn(t, dir, "checkout", "-q", "-b", "abandoned")
	if err := os.WriteFile(filepath.Join(dir, "dead.txt"), []byte("never lands\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "abandoned work")
	dead := gitIn(t, dir, "rev-parse", "HEAD")
	gitIn(t, dir, "checkout", "-q", "main")

	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDB, err := db.OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close() //nolint:errcheck
	if err := db.InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	cps := []db.CheckpointRow{{ID: "cp1", GitSHA: dead, GitBranch: "abandoned"}}

	// First pass: unmerged, so withheld — and the verdict is recorded.
	if got := filterMerged(dataDB, dir, cps); len(got) != 0 {
		t.Fatalf("unmerged work reached the wire: %+v", got)
	}
	tip := gitIn(t, dir, "rev-parse", "main")
	cached, err := db.LoadMergeGateVerdicts(dataDB, tip)
	if err != nil {
		t.Fatalf("LoadMergeGateVerdicts: %v", err)
	}
	if len(cached) == 0 {
		t.Error("nothing cached — the same probes will run again on every push, forever, for work that will never merge")
	}
	for key, shareable := range cached {
		if shareable {
			t.Errorf("cached a positive verdict for unmerged work (%s)", key)
		}
	}

	// Second pass at the same tip reuses it, and must reach the same answer.
	if got := filterMerged(dataDB, dir, cps); len(got) != 0 {
		t.Fatalf("cached run let unmerged work through: %+v", got)
	}

	// The branch merges. The mainline tip moves, so the stale "no" must not
	// survive: a cache keyed only on the checkpoint would hold this back for
	// good.
	gitIn(t, dir, "merge", "-q", "--no-ff", "-m", "merge abandoned", "abandoned")
	got := filterMerged(dataDB, dir, cps)
	if len(got) != 1 {
		t.Errorf("merged work is still withheld after the mainline moved — the verdict must be keyed to the target tip, not just the checkpoint")
	}

	// Old-tip rows are pruned rather than accumulating one set per commit.
	newTip := gitIn(t, dir, "rev-parse", "main")
	if old, err := db.LoadMergeGateVerdicts(dataDB, tip); err == nil && len(old) != 0 && tip != newTip {
		t.Errorf("verdicts for the previous tip were kept (%d rows); the table should stay proportional to checkpoints, not to mainline history", len(old))
	}
}

// TestFilterMerged_ReleasesOnSamePushThatMerges reproduces the exact
// sequence a pre-push hook runs: a feature branch merges into local main,
// then `git push` fires the hook *before* transferring anything, so
// origin/main (the remote-tracking ref) is still one push behind. The
// merged-only gate must judge against local main — which the push about to
// happen will make true on the remote — not the stale remote-tracking ref,
// or a checkpoint from a branch merged and pushed in one motion ships only
// on the *next*, unrelated push instead of "once the branch merges."
func TestFilterMerged_ReleasesOnSamePushThatMerges(t *testing.T) {
	t.Parallel()
	origin := t.TempDir()
	gitIn(t, origin, "init", "-q", "--bare")

	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	gitIn(t, dir, "clone", "-q", origin, ".")
	gitIn(t, dir, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "base")
	gitIn(t, dir, "push", "-q", "-u", "origin", "main")

	// Feature branch, merged locally but never pushed on its own — so
	// origin/main never advances to include it directly.
	gitIn(t, dir, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "feat.txt"), []byte("feature work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "feature work")
	featureSHA := gitIn(t, dir, "rev-parse", "HEAD")
	gitIn(t, dir, "checkout", "-q", "main")
	gitIn(t, dir, "merge", "-q", "--no-ff", "-m", "merge feature", "feature")

	// origin/main is deliberately NOT updated here — this is the pre-push
	// hook's vantage point: local main just advanced, the push that will
	// catch origin/main up hasn't happened yet.
	if out := gitIn(t, dir, "rev-parse", "origin/main"); out == gitIn(t, dir, "rev-parse", "main") {
		t.Fatal("test setup broken: origin/main should still be behind local main")
	}

	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDB, err := db.OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close() //nolint:errcheck
	if err := db.InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	cps := []db.CheckpointRow{{ID: "cp1", GitSHA: featureSHA, GitBranch: "feature"}}
	got := filterMerged(dataDB, dir, cps)
	if len(got) != 1 {
		t.Errorf("feature checkpoint held back even though it merged into local main ahead of this push — "+
			"the gate judged against the stale origin/main instead of the local branch about to be pushed (got %+v)", got)
	}
}

// TestPreferLocalIfAhead_DivergedLocalFallsBackToRemote pins the fail-closed
// side: preferLocalIfAhead must not upgrade to a local branch that has
// diverged from (not simply advanced past) the remote-tracking ref — that
// would judge merges against state the remote has never actually seen.
func TestPreferLocalIfAhead_DivergedLocalFallsBackToRemote(t *testing.T) {
	t.Parallel()
	origin := t.TempDir()
	gitIn(t, origin, "init", "-q", "--bare")

	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	gitIn(t, dir, "clone", "-q", origin, ".")
	gitIn(t, dir, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "base")
	gitIn(t, dir, "push", "-q", "-u", "origin", "main")

	// Push a second commit from a fresh clone so origin/main advances past
	// what this checkout has fetched — this checkout's local main and
	// origin/main have now diverged (neither is an ancestor of the other),
	// which must not be "upgraded" to local.
	otherClone := t.TempDir()
	gitIn(t, otherClone, "clone", "-q", origin, ".")
	gitIn(t, otherClone, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(otherClone, "other.txt"), []byte("someone else's work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, otherClone, "add", "-A")
	gitIn(t, otherClone, "commit", "-q", "-m", "someone else's commit")
	gitIn(t, otherClone, "push", "-q", "origin", "main")

	// This checkout fetches the teammate's push, so the divergence below is
	// locally visible (origin/main really did move to a commit dir doesn't
	// have) rather than merely stale — the guard can only defend against
	// what local git state actually knows about.
	gitIn(t, dir, "fetch", "-q", "origin")

	// dir's own commit sits on the old base, independent of what origin/main
	// now points at — a genuine divergence.
	if err := os.WriteFile(filepath.Join(dir, "mine.txt"), []byte("my own commit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "my commit")

	got := preferLocalIfAhead(dir, "origin/main")
	if got != "origin/main" {
		t.Errorf("preferLocalIfAhead(%q) = %q, want origin/main unchanged — local has diverged, not advanced, so upgrading it "+
			"would judge merges against work the remote has never seen", "origin/main", got)
	}
}
