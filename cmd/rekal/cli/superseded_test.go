package cli

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
)

// TestSupersededNomicModels_RetiredIdsAreRegistered pins the upgrade path.
//
// A model id is retired whenever anything that changes the vectors changes —
// the context window, or the document scheme fed to the embedder. An id that is
// retired but not registered here reads to recall as "some unknown model",
// which produces a message about a missing embedding config: the wrong
// diagnosis, and one the user cannot act on. Registered, it is recognised as an
// upgrade to finish, and recall refills the layer in the background.
func TestSupersededNomicModels_RetiredIdsAreRegistered(t *testing.T) {
	t.Parallel()

	for _, retired := range []string{
		"nomic-v1.5",     // 2048-token window
		"nomic-v1.5-c8k", // whole-transcript document
	} {
		if !isSupersededNomicModel(retired) {
			t.Errorf("%q is not registered as superseded — an index carrying it will be told its embedding config is missing rather than that its vectors need refilling", retired)
		}
	}
}

// TestSupersededNomicModels_CurrentIsNotRetired guards the mistake that would
// make every store re-embed on every recall: listing the live id.
func TestSupersededNomicModels_CurrentIsNotRetired(t *testing.T) {
	t.Parallel()

	if isSupersededNomicModel(nomic.ModelName) {
		t.Fatalf("the current model %q is listed as superseded — every recall would declare a healthy index stale and respawn embedding forever", nomic.ModelName)
	}
}
