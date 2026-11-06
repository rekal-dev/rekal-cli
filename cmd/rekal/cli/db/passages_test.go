package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildPassages_PreservesEverySpeakerAndLongTurn(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("证据", 3000) + " final corrected decision"
	turns := []PassageTurn{{0, "human", "original proposal", "2026-11-06T10:00:00Z"}, {1, "assistant", long, "2026-11-06T10:01:00Z"}, {2, "human_steering", "reject the earlier proposal", "2026-11-06T10:02:00Z"}}
	passages := BuildPassages("s1", turns)
	coverage := make([]bool, len(long))
	var combined strings.Builder
	for _, p := range passages {
		if len(p.Content) > passageBytes || !utf8.ValidString(p.Content) {
			t.Fatal("invalid or over-budget passage")
		}
		combined.WriteString(p.Content)
		if p.StartTurn <= 1 && p.EndTurn >= 1 {
			lo, hi := 0, len(long)
			if p.StartTurn == 1 {
				lo = p.StartByte
			}
			if p.EndTurn == 1 {
				hi = p.EndByte
			}
			for i := lo; i < hi; i++ {
				coverage[i] = true
			}
		}
	}
	for i, ok := range coverage {
		if !ok {
			t.Fatalf("assistant byte %d was dropped", i)
		}
	}
	for _, want := range []string{"role=human ", "role=assistant ", "role=human_steering ", "2026-11-06T10:01:00Z", "final corrected decision", "reject the earlier proposal"} {
		if !strings.Contains(combined.String(), want) {
			t.Fatalf("missing %q", want)
		}
	}
	if turns[1].Content != long {
		t.Fatal("source turn changed")
	}
}

func TestRefreshSessionPassages_AppendRetainsStableVectors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := InitIndexSchema(d); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO session_facets(session_id,actor_type,captured_at) VALUES ('s1','human','2026-11-06T10:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	insert := func(i int) {
		t.Helper()
		if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ($1,'s1',$2,'assistant',$3,'2026-11-06T10:00:00Z')`, fmt.Sprint(i), i, fmt.Sprintf("evidence %d", i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 20; i++ {
		insert(i)
	}
	if err := RefreshSessionPassages(d); err != nil {
		t.Fatal(err)
	}
	docs, err := QueryPassageDocuments(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 {
		t.Fatalf("passages=%d, want 3 overlapping windows", len(docs))
	}
	vectors := map[string][]float64{}
	for id := range docs {
		vectors[id] = []float64{1, 0}
	}
	if err := StorePassageEmbeddings(d, vectors, "test"); err != nil {
		t.Fatal(err)
	}
	if err := StorePassageEmbeddings(d, vectors, "test"); err != nil {
		t.Fatalf("repeat vector storage is not idempotent: %v", err)
	}
	insert(20)
	if err := RefreshSessionPassages(d, "s1"); err != nil {
		t.Fatal(err)
	}
	kept, err := QueryPassageVectors(d, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 2 {
		t.Fatalf("stable vectors=%d, want 2; only the last passage should change", len(kept))
	}
	for _, p := range kept {
		if _, ok := docs[p.ID]; !ok {
			t.Fatal("stable passage identity changed")
		}
	}
	if err := RefreshSessionPassages(d, "s1"); err != nil {
		t.Fatal(err)
	}
	current, err := QueryPassageDocuments(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 3 {
		t.Fatalf("append/retry created %d passages", len(current))
	}
}
