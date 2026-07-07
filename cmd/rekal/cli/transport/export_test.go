package transport

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

func TestShareableCheckpoints(t *testing.T) {
	t.Parallel()

	cps := []db.CheckpointRow{
		{ID: "a", GitSHA: "merged1"},
		{ID: "b", GitSHA: "unmerged"},
		{ID: "c", GitSHA: "merged2"},
		{ID: "d", GitSHA: ""}, // no sha — never shareable
	}
	merged := map[string]bool{"merged1": true, "merged2": true}

	got := shareableCheckpoints(cps, func(sha string) bool { return merged[sha] })

	if len(got) != 2 {
		t.Fatalf("expected 2 shareable, got %d: %+v", len(got), got)
	}
	if got[0].ID != "a" || got[1].ID != "c" {
		t.Fatalf("wrong checkpoints kept: %+v", got)
	}

	// Input must be untouched — held-back rows stay unexported for the next push.
	if len(cps) != 4 {
		t.Fatalf("input slice was mutated: len=%d", len(cps))
	}
}

func TestShareableCheckpoints_NoneMerged(t *testing.T) {
	t.Parallel()
	cps := []db.CheckpointRow{{ID: "a", GitSHA: "x"}, {ID: "b", GitSHA: "y"}}
	got := shareableCheckpoints(cps, func(string) bool { return false })
	if len(got) != 0 {
		t.Fatalf("expected nothing shareable, got %d", len(got))
	}
}
