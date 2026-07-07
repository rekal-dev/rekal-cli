package transport

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// fakes for the pure gate logic — the git-backed predicates are exercised in
// gitx's own tests and the cli end-to-end gate tests.
type gateFakes struct {
	ancestors map[string]bool     // sha → on mainline
	squashed  map[string]bool     // sha → cumulative diff landed as squash
	tips      map[string]string   // branch → local tip sha
	lineage   map[string][]string // tip sha → shas reachable from it
}

func (f gateFakes) run(cps []db.CheckpointRow) []db.CheckpointRow {
	return shareableCheckpoints(cps,
		func(sha string) bool { return f.ancestors[sha] },
		func(sha string) bool { return f.squashed[sha] },
		func(branch string) string { return f.tips[branch] },
		func(sha, tip string) bool {
			if sha == tip {
				return true
			}
			for _, a := range f.lineage[tip] {
				if a == sha {
					return true
				}
			}
			return false
		},
	)
}

func cpIDs(cps []db.CheckpointRow) []string {
	out := make([]string, len(cps))
	for i, cp := range cps {
		out[i] = cp.ID
	}
	return out
}

func TestShareableCheckpoints_AncestorGate(t *testing.T) {
	t.Parallel()

	cps := []db.CheckpointRow{
		{ID: "a", GitSHA: "merged1", GitBranch: "main"},
		{ID: "b", GitSHA: "unmerged", GitBranch: "feat"},
		{ID: "c", GitSHA: "merged2", GitBranch: "main"},
		{ID: "d", GitSHA: "", GitBranch: "main"}, // no sha — never shareable
	}
	f := gateFakes{ancestors: map[string]bool{"merged1": true, "merged2": true}}

	got := f.run(cps)
	if g := cpIDs(got); len(g) != 2 || g[0] != "a" || g[1] != "c" {
		t.Fatalf("kept %v, want [a c]", g)
	}
	// Input must be untouched — held-back rows stay unexported for the next push.
	if len(cps) != 4 {
		t.Fatalf("input slice was mutated: len=%d", len(cps))
	}
}

func TestShareableCheckpoints_NoneMerged(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{{ID: "a", GitSHA: "x"}, {ID: "b", GitSHA: "y"}}
	got := gateFakes{}.run(cps)
	if len(got) != 0 {
		t.Fatalf("expected nothing shareable, got %v", cpIDs(got))
	}
}

// TestShareableCheckpoints_SquashDirect: a checkpoint at the squashed branch's
// final state is released by the squash signal alone.
func TestShareableCheckpoints_SquashDirect(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{
		{ID: "tip", GitSHA: "feat-tip", GitBranch: "feat"},
		{ID: "dead", GitSHA: "dead-tip", GitBranch: "spike"},
	}
	f := gateFakes{squashed: map[string]bool{"feat-tip": true}}
	got := f.run(cps)
	if g := cpIDs(got); len(g) != 1 || g[0] != "tip" {
		t.Fatalf("kept %v, want [tip]", g)
	}
}

// TestShareableCheckpoints_SquashReleasesMidBranch: mid-branch checkpoints are
// released when a later checkpoint on the same branch proved the squash and
// they are ancestors of it — and only then.
func TestShareableCheckpoints_SquashReleasesMidBranch(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{
		{ID: "mid", GitSHA: "feat-c1", GitBranch: "feat"},
		{ID: "tip", GitSHA: "feat-c2", GitBranch: "feat"},
		{ID: "other", GitSHA: "spike-c1", GitBranch: "spike"}, // unrelated branch
	}
	f := gateFakes{
		squashed: map[string]bool{"feat-c2": true},
		lineage:  map[string][]string{"feat-c2": {"feat-c1"}},
	}
	got := f.run(cps)
	if g := cpIDs(got); len(g) != 2 || g[0] != "mid" || g[1] != "tip" {
		t.Fatalf("kept %v, want [mid tip]", g)
	}
}

// TestShareableCheckpoints_SquashViaLocalTip: no checkpoint exists at the
// branch tip, but the surviving local branch tip proves the squash and
// releases the branch's checkpoints beneath it.
func TestShareableCheckpoints_SquashViaLocalTip(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{
		{ID: "mid", GitSHA: "feat-c1", GitBranch: "feat"},
	}
	f := gateFakes{
		squashed: map[string]bool{"feat-tip": true},
		tips:     map[string]string{"feat": "feat-tip"},
		lineage:  map[string][]string{"feat-tip": {"feat-c1"}},
	}
	got := f.run(cps)
	if g := cpIDs(got); len(g) != 1 || g[0] != "mid" {
		t.Fatalf("kept %v, want [mid]", g)
	}
}

// TestShareableCheckpoints_SquashDoesNotLeakSiblings: a proven squash on one
// branch must not release checkpoints that are not ancestors of the proven
// point, even on the same branch name.
func TestShareableCheckpoints_SquashDoesNotLeakSiblings(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{
		{ID: "tip", GitSHA: "feat-c2", GitBranch: "feat"},
		{ID: "later", GitSHA: "feat-c3", GitBranch: "feat"}, // after the squash point
	}
	f := gateFakes{
		squashed: map[string]bool{"feat-c2": true},
		lineage:  map[string][]string{"feat-c2": {"feat-c1"}}, // c3 not reachable
	}
	got := f.run(cps)
	if g := cpIDs(got); len(g) != 1 || g[0] != "tip" {
		t.Fatalf("kept %v, want [tip] only — feat-c3 is not part of the landed state", g)
	}
}
