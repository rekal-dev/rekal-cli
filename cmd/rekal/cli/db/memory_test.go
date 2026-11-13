package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

func TestMemoryHistoryAndProjection(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(StoreDir(root), 0700); err != nil {
		t.Fatal(err)
	}
	d, err := OpenData(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := InitDataSchema(d); err != nil {
		t.Fatal(err)
	}
	source := memory.Source{Name: "native", Adapter: "markdown", Owner: "alice", Profile: "profile"}
	files := []memory.File{{Path: "a.md", Content: "first", ModifiedAt: "2026-01-01T00:00:00Z"}}
	first := MemoryChanges(nil, source, files, "cp1", "sha", "2026-11-14T01:00:00Z")
	if err := InsertMemoryRevisions(d, first); err != nil {
		t.Fatal(err)
	}
	if err := InsertMemoryRevisions(d, first); err != nil {
		t.Fatal(err)
	}
	previous, err := LatestMemory(d, source.ID())
	if err != nil {
		t.Fatal(err)
	}
	files[0].ModifiedAt = "2026-11-14T00:00:00Z"
	if got := MemoryChanges(previous, source, files, "cp2", "sha", "2026-11-14T02:00:00Z"); len(got) != 0 {
		t.Fatal("touching unchanged text created a revision")
	}
	files[0].Content = "second"
	second := MemoryChanges(previous, source, files, "cp2", "sha", "2026-11-14T02:00:00Z")
	if err := InsertMemoryRevisions(d, second); err != nil {
		t.Fatal(err)
	}
	previous, err = LatestMemory(d, source.ID())
	if err != nil {
		t.Fatal(err)
	}
	deleted := MemoryChanges(previous, source, nil, "cp3", "sha", "2026-11-14T03:00:00Z")
	if err := InsertMemoryRevisions(d, deleted); err != nil {
		t.Fatal(err)
	}
	// Reappearance of old text is a new observation, not deduplicated out of history.
	previous, err = LatestMemory(d, source.ID())
	if err != nil {
		t.Fatal(err)
	}
	files[0].Content = "first"
	returned := MemoryChanges(previous, source, files, "cp4", "sha", "2026-11-14T04:00:00Z")
	if len(returned) != 1 || returned[0].ID == first[0].ID {
		t.Fatal("reappearance lost")
	}
	if err := InsertMemoryRevisions(d, returned); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	for range 2 {
		index, err := OpenIndexAt(filepath.Join(t.TempDir(), "index.db"))
		if err != nil {
			t.Fatal(err)
		}
		if err := InitIndexSchema(index); err != nil {
			t.Fatal(err)
		}
		if err := PopulateIndex(index, root); err != nil {
			t.Fatal(err)
		}
		var id string
		if err := index.QueryRow("SELECT id FROM memory_documents").Scan(&id); err != nil || id != returned[0].ID {
			t.Fatalf("current projection %q %v", id, err)
		}
		var n int
		if err := index.QueryRow("SELECT count(*) FROM memory_archive").Scan(&n); err != nil || n != 4 {
			t.Fatalf("history %d %v", n, err)
		}
		if err := PopulateIndexIncremental(index, root, nil, ""); err != nil {
			t.Fatal(err)
		}
		if err := index.QueryRow("SELECT count(*) FROM memory_documents").Scan(&n); err != nil || n != 1 {
			t.Fatalf("duplicate projection %d %v", n, err)
		}
		_ = index.Close()
	}
}
