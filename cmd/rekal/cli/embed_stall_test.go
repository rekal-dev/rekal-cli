package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// failingEmbedder stands in for a backend that is reachable at construction and
// fails every call afterwards — a daemon that exited, an HTTP endpoint that
// started refusing, an expired key. It counts calls so the test can prove the
// loop stopped instead of spinning.
type failingEmbedder struct{ calls int }

func (f *failingEmbedder) ModelName() string { return "test-model" }
func (f *failingEmbedder) Close()            {}
func (f *failingEmbedder) EmbedSessions(map[string]string) (map[string][]float64, error) {
	f.calls++
	return nil, errors.New("write unix: broken pipe")
}

// TestEmbedBite_ReportsNoProgressWhenTheBackendFails pins the signal the loop
// needs to tell a stall from progress.
//
// Both halves of a bite warn and swallow their errors, and knowledgeMore is
// recomputed from the database — so on a backend that fails every call, "work
// remains" stays true forever while nothing is ever written. The loop ran 353
// identical passes in one measured run, printing a warning each time and
// embedding nothing, and index/sync spawn it in the background where it burns a
// core until something kills it.
func TestEmbedBite_ReportsNoProgressWhenTheBackendFails(t *testing.T) {
	gitRoot := t.TempDir()
	gitRoot, err := filepath.EvalSymlinks(gitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gitRoot, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("knowledge schema: %v", err)
	}
	// One session with content, so the bite has something to try to embed.
	if _, err := indexDB.Exec(
		`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
		 VALUES ('t1', 's1', 0, 'human', 'a conversation about the export gate', '')`,
	); err != nil {
		t.Fatalf("seed turn: %v", err)
	}
	indexDB.Close() //nolint:errcheck

	emb := &failingEmbedder{}
	var out strings.Builder
	_, wrote, err := embedBite(&out, gitRoot, 1, emb)
	if err != nil {
		t.Fatalf("embedBite returned a hard error: %v", err)
	}
	if wrote {
		t.Error("embedBite reported progress though every embed call failed — the loop cannot tell a stall from work and will spin")
	}
	if emb.calls == 0 {
		t.Skip("bite never reached the embedder; fixture does not exercise the path")
	}
}

// TestStoredVectorCount_CountsBothLayers pins the measure the stall check rests
// on: if it silently returned a constant, every pass would look like a stall
// (or none would).
func TestStoredVectorCount_CountsBothLayers(t *testing.T) {
	gitRoot := t.TempDir()
	gitRoot, err := filepath.EvalSymlinks(gitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gitRoot, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := db.InitIndexSchema(indexDB); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		t.Fatalf("knowledge schema: %v", err)
	}

	if got := storedVectorCount(indexDB); got != 0 {
		t.Fatalf("empty index reports %d vectors, want 0", got)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO session_embeddings (session_id, model, embedding, generated_at)
		 VALUES ('s1', 'm', [0.1]::FLOAT[], now())`,
	); err != nil {
		t.Fatalf("store a vector: %v", err)
	}
	if got := storedVectorCount(indexDB); got != 1 {
		t.Errorf("after one stored vector the count is %d, want 1 — the stall check reads this number", got)
	}
}
