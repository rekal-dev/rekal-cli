package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPopulateIndexIncremental_PreservesRowsAndHistoricalFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := OpenData(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := InitDataSchema(data); err != nil {
		t.Fatal(err)
	}
	if err := InsertSession(data, "s1", "", "hash", "human", "", "dev@example.com", "main", "2026-11-06T10:00:00Z", "claude"); err != nil {
		t.Fatal(err)
	}
	if err := data.Close(); err != nil {
		t.Fatal(err)
	}
	index, err := OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	if err := InitIndexSchema(index); err != nil {
		t.Fatal(err)
	}
	exec := func(d *sql.DB, q string, args ...interface{}) {
		t.Helper()
		if _, err := d.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	var firstRowID int64
	for i := 0; i < 4; i++ {
		data, err = OpenData(root)
		if err != nil {
			t.Fatal(err)
		}
		checkpoint := fmt.Sprintf("c%d", i)
		if err := InsertTurn(data, fmt.Sprintf("t%d", i), "s1", i, "human", fmt.Sprintf("new evidence %d", i), "2026-11-06T10:00:00Z"); err != nil {
			t.Fatal(err)
		}
		if err := InsertToolCall(data, fmt.Sprintf("tool%d", i), "s1", i, "Write", fmt.Sprintf("file%d.go", i), ""); err != nil {
			t.Fatal(err)
		}
		if err := InsertCheckpoint(data, checkpoint, fmt.Sprintf("sha%d", i), "main", "dev@example.com", fmt.Sprintf("2026-11-06T10:00:0%dZ", i), "human", ""); err != nil {
			t.Fatal(err)
		}
		if err := InsertCheckpointSession(data, checkpoint, "s1"); err != nil {
			t.Fatal(err)
		}
		if err := InsertFileTouched(data, fmt.Sprintf("f%d", i), checkpoint, fmt.Sprintf("file%d.go", i), "M"); err != nil {
			t.Fatal(err)
		}
		if err := data.Close(); err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			exec(index, `INSERT INTO session_embeddings (session_id,embedding,model,generated_at) VALUES ('s1',[1.0,2.0]::FLOAT[],'stale-test',now())`)
		}
		if err := PopulateIndexIncremental(index, root, []string{"s1"}, checkpoint); err != nil {
			t.Fatal(err)
		}
		var rowID int64
		if err := index.QueryRow(`SELECT rowid FROM turns_ft WHERE id='t0'`).Scan(&rowID); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstRowID = rowID
		} else if rowID != firstRowID {
			t.Fatalf("an unchanged turn was rewritten: rowid %d -> %d", firstRowID, rowID)
		}
		for _, table := range []string{"turns_ft", "tool_calls_index", "files_index"} {
			var count int
			if err := index.QueryRow("SELECT count(*) FROM " + table + " WHERE session_id='s1'").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != i+1 {
				t.Fatalf("%s rows=%d, want %d; history lost or duplicated", table, count, i+1)
			}
		}
		var stale int
		if err := index.QueryRow(`SELECT count(*) FROM session_embeddings WHERE model='stale-test'`).Scan(&stale); err != nil {
			t.Fatal(err)
		}
		if stale != 0 {
			t.Fatal("stale whole-session vector survived growth")
		}
	}
	// Retrying an already-indexed checkpoint must not duplicate immutable rows.
	if err := PopulateIndexIncremental(index, root, []string{"s1"}, "c3"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := index.QueryRow(`SELECT count(*) FROM files_index`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("retry duplicated historical file links: %d", count)
	}
	var turns, tools, files int
	if err := index.QueryRow(`SELECT turn_count,tool_call_count,file_count FROM session_facets WHERE session_id='s1'`).Scan(&turns, &tools, &files); err != nil {
		t.Fatal(err)
	}
	if turns != 4 || tools != 4 || files != 4 {
		t.Fatalf("facets stale: turns=%d tools=%d files=%d", turns, tools, files)
	}
}
