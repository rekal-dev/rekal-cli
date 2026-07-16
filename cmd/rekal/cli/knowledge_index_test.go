package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

func commitFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	runGit(t, dir, "add", path)
	runGit(t, dir, "commit", "-m", "update "+path)
}

// TestRefreshKnowledge covers the knowledge layer's build/refresh cycle
// against a real git repo: full build at first refresh, watermark short-
// circuit, surgical re-chunk on change, and row removal on delete.
func TestRefreshKnowledge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "dev@example.com")
	runGit(t, dir, "config", "user.name", "Dev")
	commitFile(t, dir, "docs/auth.md", "# Auth\n\nTokens rotate on every use.\n")
	commitFile(t, dir, "main.go", "package main\n")
	runGit(t, dir, "branch", "-M", "main")

	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatalf("mkdir .rekal: %v", err)
	}
	indexDB, err := db.OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	t.Cleanup(func() { indexDB.Close() })
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}

	countChunks := func(path string) int {
		t.Helper()
		var n int
		if err := indexDB.QueryRow(
			`SELECT count(*) FROM knowledge_chunks WHERE path = $1`, path).Scan(&n); err != nil {
			t.Fatalf("count chunks: %v", err)
		}
		return n
	}

	// Full build: prose file chunked, code file ignored.
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		t.Fatalf("refreshKnowledge (full): %v", err)
	}
	if n := countChunks("docs/auth.md"); n != 1 {
		t.Fatalf("docs/auth.md chunks = %d, want 1", n)
	}
	if n := countChunks("main.go"); n != 0 {
		t.Fatalf("main.go must not be indexed, got %d chunk(s)", n)
	}

	// Watermark: second call with unmoved HEAD is a no-op.
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		t.Fatalf("refreshKnowledge (steady state): %v", err)
	}

	// Change + add: only the changed/new files re-chunk.
	commitFile(t, dir, "docs/auth.md", "# Auth\n\nTokens rotate hourly.\n\n## Failure modes\n\n401 on expiry.\n")
	commitFile(t, dir, "docs/deploy.md", "# Deploy\n\nShip from main.\n")
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		t.Fatalf("refreshKnowledge (incremental): %v", err)
	}
	if n := countChunks("docs/auth.md"); n != 2 {
		t.Fatalf("docs/auth.md chunks after change = %d, want 2", n)
	}
	if n := countChunks("docs/deploy.md"); n != 1 {
		t.Fatalf("docs/deploy.md chunks = %d, want 1", n)
	}

	// Paths with spaces/UTF-8 must index too — ls-tree is parsed with -z, so
	// git's C-quoting can never hide a file from the change detector.
	commitFile(t, dir, "docs/über notes.md", "# Über\n\nNotes with a spaced, non-ASCII path.\n")
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		t.Fatalf("refreshKnowledge (special path): %v", err)
	}
	if n := countChunks("docs/über notes.md"); n != 1 {
		t.Fatalf("special-char path chunks = %d, want 1", n)
	}

	// Delete: rows drop with the file.
	runGit(t, dir, "rm", "docs/deploy.md")
	runGit(t, dir, "commit", "-m", "remove deploy doc")
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		t.Fatalf("refreshKnowledge (delete): %v", err)
	}
	if n := countChunks("docs/deploy.md"); n != 0 {
		t.Fatalf("deleted file still indexed: %d chunk(s)", n)
	}

	// The FTS index over the final chunk set answers queries.
	rows, err := indexDB.Query(`
		SELECT kc.path, fts_main_knowledge_chunks.match_bm25(kc.id, 'failure modes') AS score
		FROM knowledge_chunks kc WHERE score IS NOT NULL`)
	if err != nil {
		t.Fatalf("knowledge FTS query: %v", err)
	}
	defer rows.Close() //nolint:errcheck
	found := false
	for rows.Next() {
		var path string
		var score float64
		if err := rows.Scan(&path, &score); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if path == "docs/auth.md" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if !found {
		t.Fatal("FTS index should match the re-chunked section")
	}
}
