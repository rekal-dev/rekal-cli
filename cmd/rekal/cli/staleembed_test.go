package cli

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

// TestStaleEmbeddedVectors_FutureBumpNeedsNoList is the reason this replaced a
// hand-maintained set of retired ids.
//
// Vectors are keyed by model id, and anything that changes what the embedder
// produces retires the id. With a list, retiring one meant remembering to add
// it; forget, and recall tells the user their embedding config is missing —
// the wrong diagnosis, unactionable, and silent. The comparison is now
// mechanical, so an id nobody has ever seen is still recognised as stale.
func TestStaleEmbeddedVectors_FutureBumpNeedsNoList(t *testing.T) {
	t.Parallel()

	if !staleEmbeddedVectors(search.EmbedderBackendEmbedded, "nomic-v9.9-not-in-any-list") {
		t.Error("an unrecognised embedded model id was not treated as stale — a future bump would ship without its upgrade path again")
	}
	if legacyEmbeddedModels["nomic-v9.9-not-in-any-list"] {
		t.Fatal("fixture leaked into the legacy set")
	}
}

// TestStaleEmbeddedVectors_CurrentIsFresh guards the inverse mistake, which is
// worse: a healthy index declared stale on every recall would respawn embedding
// forever.
func TestStaleEmbeddedVectors_CurrentIsFresh(t *testing.T) {
	t.Parallel()

	if staleEmbeddedVectors(search.EmbedderBackendEmbedded, nomic.ModelName) {
		t.Fatalf("current model %q reported stale — every recall would respawn embedding", nomic.ModelName)
	}
}

// TestStaleEmbeddedVectors_HTTPModelIsNotOurUpgrade pins the distinction the
// backend was recorded for.
//
// An HTTP model id that no longer matches means the embedding *config* was
// removed, not that this binary's own scheme moved on. Re-embedding locally
// would answer a question nobody asked — and silently replace a store's vectors
// with ones from a different model.
func TestStaleEmbeddedVectors_HTTPModelIsNotOurUpgrade(t *testing.T) {
	t.Parallel()

	if staleEmbeddedVectors(search.EmbedderBackendHTTP, "text-embedding-3-small") {
		t.Error("an HTTP-embedded index was treated as a local upgrade — that would re-embed someone's store with a different model behind their back")
	}
	// Even when the id happens to look like ours.
	if staleEmbeddedVectors(search.EmbedderBackendHTTP, "nomic-embed-text-v1.5") {
		t.Error("an HTTP index serving a nomic-named model was treated as a local upgrade")
	}
}

// TestStaleEmbeddedVectors_LegacyIndexStillUpgrades keeps the stores that
// predate embed_backend working: they carry no backend, so the closed legacy
// set is the only thing that can answer for them.
func TestStaleEmbeddedVectors_LegacyIndexStillUpgrades(t *testing.T) {
	t.Parallel()

	for _, old := range []string{"nomic-v1.5", "nomic-v1.5-c8k"} {
		if !staleEmbeddedVectors("", old) {
			t.Errorf("legacy index carrying %q was not recognised as stale", old)
		}
	}
	// An unknown id with no backend is not assumed to be ours — it may have
	// come from an HTTP backend configured before the column existed.
	if staleEmbeddedVectors("", "text-embedding-3-small") {
		t.Error("an unknown model on a legacy index was claimed as a local upgrade")
	}
}
