package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

type recordingPassageEmbedder struct{ docs map[string]string }

func (r *recordingPassageEmbedder) ModelName() string { return "passage-test" }
func (r *recordingPassageEmbedder) Close()            {}
func (r *recordingPassageEmbedder) EmbedSessions(docs map[string]string) (map[string][]float64, error) {
	out := map[string][]float64{}
	for id, text := range docs {
		r.docs[id] = text
		out[id] = []float64{1, 0}
	}
	return out, nil
}

func TestBuildPassageEmbeddings_BudgetAndStableAppend(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0755); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := db.InitIndexSchema(d); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO session_facets(session_id,actor_type,captured_at) VALUES ('session-a','human','2026-11-06T10:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ($1,'session-a',$2,'assistant',$3,'2026-11-06T10:00:00Z')`, fmt.Sprint(i), i, fmt.Sprintf("original evidence %d", i)); err != nil {
			t.Fatal(err)
		}
	}
	r := &recordingPassageEmbedder{docs: map[string]string{}}
	for i := 0; i < 3; i++ {
		more, err := buildPassageEmbeddings(d, nil, io.Discard, root, 1, r)
		if err != nil {
			t.Fatal(err)
		}
		if more != (i < 2) {
			t.Fatalf("pass %d remaining=%v", i, more)
		}
	}
	if len(r.docs) != 3 {
		t.Fatalf("embedded=%d, want 3", len(r.docs))
	}
	for _, text := range r.docs {
		if !strings.Contains(text, "role=assistant") || !strings.Contains(text, "ts=2026-11-06") {
			t.Fatal("speaker/time metadata missing")
		}
	}
	if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ('20','session-a',20,'assistant','late correction','2026-11-06T11:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := buildPassageEmbeddings(d, nil, io.Discard, root, 0, r); err != nil {
		t.Fatal(err)
	}
	if len(r.docs) != 4 {
		t.Fatalf("append re-embedded stable passages: %d documents", len(r.docs))
	}
	vectors, err := db.QueryPassageVectors(d, "passage-test")
	if err != nil || len(vectors) != 3 {
		t.Fatalf("vectors=%d err=%v", len(vectors), err)
	}
}
