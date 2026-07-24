package cli

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/spf13/cobra"
)

// knowledgeTestRepo is a temp git repo + index DB ready for refreshKnowledge.
type knowledgeTestRepo struct {
	dir     string
	indexDB *sql.DB
}

func newKnowledgeTestRepo(t *testing.T) *knowledgeTestRepo {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "dev@example.com")
	runGit(t, dir, "config", "user.name", "Dev")
	runGit(t, dir, "config", "gc.auto", "0")
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
	return &knowledgeTestRepo{dir: dir, indexDB: indexDB}
}

func (r *knowledgeTestRepo) refresh(t *testing.T) {
	t.Helper()
	if err := refreshKnowledge(nil, r.indexDB, r.dir); err != nil {
		t.Fatalf("refreshKnowledge: %v", err)
	}
}

func (r *knowledgeTestRepo) countPath(t *testing.T, path string) int {
	t.Helper()
	var n int
	if err := r.indexDB.QueryRow(
		`SELECT count(*) FROM knowledge_chunks WHERE path = $1`, path).Scan(&n); err != nil {
		t.Fatalf("count chunks %s: %v", path, err)
	}
	return n
}

func (r *knowledgeTestRepo) chunkRow(t *testing.T, path string) (content, blobSHA, contentHash, id string) {
	t.Helper()
	err := r.indexDB.QueryRow(`
		SELECT content, blob_sha, content_hash, id FROM knowledge_chunks
		WHERE path = $1 ORDER BY start_line LIMIT 1`, path).
		Scan(&content, &blobSHA, &contentHash, &id)
	if err != nil {
		t.Fatalf("chunk row %s: %v", path, err)
	}
	return content, blobSHA, contentHash, id
}

func (r *knowledgeTestRepo) watermark(t *testing.T) string {
	t.Helper()
	v, ok, err := db.ReadIndexState(r.indexDB, db.KnowledgeHeadKey)
	if err != nil {
		t.Fatalf("ReadIndexState: %v", err)
	}
	if !ok {
		t.Fatal("knowledge watermark missing")
	}
	return v
}

func (r *knowledgeTestRepo) ftsHits(t *testing.T, query string) []string {
	t.Helper()
	rows, err := r.indexDB.Query(`
		SELECT kc.path, fts_main_knowledge_chunks.match_bm25(kc.id, $1) AS score
		FROM knowledge_chunks kc WHERE score IS NOT NULL`, query)
	if err != nil {
		t.Fatalf("knowledge FTS: %v", err)
	}
	defer rows.Close() //nolint:errcheck
	var paths []string
	for rows.Next() {
		var path string
		var score float64
		if err := rows.Scan(&path, &score); err != nil {
			t.Fatalf("scan FTS: %v", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("FTS rows: %v", err)
	}
	return paths
}

// TestRefreshKnowledge_UpdateReplacesContentAndBlobSHA verifies an edited
// prose file is re-chunked: content, blob_sha, and FTS all track HEAD.
func TestRefreshKnowledge_UpdateReplacesContentAndBlobSHA(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/auth.md", "# Auth\n\nTokens rotate on every use.\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)

	oldContent, oldBlob, _, _ := r.chunkRow(t, "docs/auth.md")
	if !strings.Contains(oldContent, "every use") {
		t.Fatalf("initial content = %q, want 'every use'", oldContent)
	}
	head1 := gitx.HeadSHA(r.dir)
	if got := r.watermark(t); got != head1 {
		t.Fatalf("watermark = %s, want %s", got, head1)
	}

	commitFile(t, r.dir, "docs/auth.md", "# Auth\n\nTokens rotate hourly with alphafresh.\n")
	r.refresh(t)

	newContent, newBlob, _, _ := r.chunkRow(t, "docs/auth.md")
	if strings.Contains(newContent, "every use") {
		t.Fatal("stale content still indexed after update")
	}
	if !strings.Contains(newContent, "alphafresh") {
		t.Fatalf("updated content = %q, want alphafresh", newContent)
	}
	if newBlob == oldBlob {
		t.Fatal("blob_sha unchanged after content edit")
	}
	if r.countPath(t, "docs/auth.md") != 1 {
		t.Fatalf("chunk count = %d, want 1", r.countPath(t, "docs/auth.md"))
	}
	head2 := gitx.HeadSHA(r.dir)
	if got := r.watermark(t); got != head2 {
		t.Fatalf("watermark = %s, want %s", got, head2)
	}
	hits := r.ftsHits(t, "alphafresh")
	if len(hits) == 0 {
		t.Fatal("FTS should match the new term")
	}
	for _, p := range hits {
		if p != "docs/auth.md" {
			t.Fatalf("unexpected FTS hit path %q", p)
		}
	}
}

// TestRefreshKnowledge_DeletePurgesChunksAndFTS verifies a removed prose file
// drops out of knowledge_chunks and the knowledge FTS index.
func TestRefreshKnowledge_DeletePurgesChunksAndFTS(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/keep.md", "# Keep\n\nStable keepterm.\n")
	commitFile(t, r.dir, "docs/gone.md", "# Gone\n\nUnique deleteterm.\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)
	if r.countPath(t, "docs/gone.md") != 1 {
		t.Fatalf("pre-delete gone.md chunks = %d", r.countPath(t, "docs/gone.md"))
	}

	runGit(t, r.dir, "rm", "docs/gone.md")
	runGit(t, r.dir, "commit", "-m", "remove gone.md")
	r.refresh(t)

	if r.countPath(t, "docs/gone.md") != 0 {
		t.Fatalf("deleted file still indexed: %d chunk(s)", r.countPath(t, "docs/gone.md"))
	}
	if r.countPath(t, "docs/keep.md") != 1 {
		t.Fatal("unrelated file was purged on delete")
	}
	if got := r.watermark(t); got != gitx.HeadSHA(r.dir) {
		t.Fatalf("watermark not advanced after delete")
	}
	if hits := r.ftsHits(t, "deleteterm"); len(hits) != 0 {
		t.Fatalf("FTS still matches deleted term: %v", hits)
	}
	if hits := r.ftsHits(t, "keepterm"); len(hits) == 0 {
		t.Fatal("FTS lost the kept file")
	}
}

// TestRefreshKnowledge_RenameMovesChunks verifies git mv: old path purged,
// new path indexed, same blob content-hash identity (embed-cache friendly).
func TestRefreshKnowledge_RenameMovesChunks(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/old.md", "# Policy\n\nRename-stable section body.\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)

	_, oldBlob, oldHash, oldID := r.chunkRow(t, "docs/old.md")
	if !strings.HasPrefix(oldID, "docs/old.md#") {
		t.Fatalf("chunk id = %q, want docs/old.md#…", oldID)
	}

	runGit(t, r.dir, "mv", "docs/old.md", "docs/new.md")
	runGit(t, r.dir, "commit", "-m", "rename policy doc")
	r.refresh(t)

	if r.countPath(t, "docs/old.md") != 0 {
		t.Fatalf("old path still indexed: %d", r.countPath(t, "docs/old.md"))
	}
	if r.countPath(t, "docs/new.md") != 1 {
		t.Fatalf("new path chunks = %d, want 1", r.countPath(t, "docs/new.md"))
	}
	newContent, newBlob, newHash, newID := r.chunkRow(t, "docs/new.md")
	if !strings.Contains(newContent, "Rename-stable") {
		t.Fatalf("renamed content lost: %q", newContent)
	}
	if newBlob != oldBlob {
		t.Fatalf("blob_sha after rename = %s, want %s (same git blob)", newBlob, oldBlob)
	}
	if newHash != oldHash {
		t.Fatalf("content_hash changed on rename: %s → %s (embed cache identity)", oldHash, newHash)
	}
	if !strings.HasPrefix(newID, "docs/new.md#") {
		t.Fatalf("chunk id = %q, want docs/new.md#…", newID)
	}
	if got := r.watermark(t); got != gitx.HeadSHA(r.dir) {
		t.Fatal("watermark not advanced after rename")
	}
}

// TestRefreshKnowledge_CodeOnlyAdvancesWatermark verifies a non-prose commit
// still moves the watermark without rewriting chunk rows.
func TestRefreshKnowledge_CodeOnlyAdvancesWatermark(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/api.md", "# API\n\nStable prose.\n")
	commitFile(t, r.dir, "main.go", "package main\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)
	_, blobBefore, hashBefore, _ := r.chunkRow(t, "docs/api.md")
	headBefore := r.watermark(t)

	commitFile(t, r.dir, "main.go", "package main\n\nfunc main() {}\n")
	r.refresh(t)

	headAfter := gitx.HeadSHA(r.dir)
	if headAfter == headBefore {
		t.Fatal("expected HEAD to move on code commit")
	}
	if got := r.watermark(t); got != headAfter {
		t.Fatalf("watermark = %s, want %s", got, headAfter)
	}
	_, blobAfter, hashAfter, _ := r.chunkRow(t, "docs/api.md")
	if blobAfter != blobBefore || hashAfter != hashBefore {
		t.Fatal("code-only commit rewrote prose chunks")
	}
	if r.countPath(t, "docs/api.md") != 1 {
		t.Fatal("unexpected chunk count after code-only commit")
	}
}

// TestRefreshKnowledge_WatermarkShortCircuitSkipsRepair verifies a same-HEAD
// refresh does not rebuild from disk — corrupted rows stay until HEAD moves.
func TestRefreshKnowledge_WatermarkShortCircuitSkipsRepair(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/a.md", "# A\n\nOriginal body.\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)

	if _, err := r.indexDB.Exec(
		`UPDATE knowledge_chunks SET content = 'poisoned' WHERE path = $1`, "docs/a.md"); err != nil {
		t.Fatalf("corrupt chunks: %v", err)
	}
	r.refresh(t) // same HEAD — must short-circuit
	content, _, _, _ := r.chunkRow(t, "docs/a.md")
	if content != "poisoned" {
		t.Fatalf("short-circuit repaired content unexpectedly: %q", content)
	}

	// Touch HEAD with a no-content prose change… actually need HEAD to move.
	// A code-only commit advances watermark and runs the blob diff; prose
	// blob unchanged so poisoned row is NOT rebuilt (diff empty for that path).
	// To force repair, change the prose blob.
	commitFile(t, r.dir, "docs/a.md", "# A\n\nRepaired body uniquetoken.\n")
	r.refresh(t)
	content, _, _, _ = r.chunkRow(t, "docs/a.md")
	if strings.Contains(content, "poisoned") || !strings.Contains(content, "uniquetoken") {
		t.Fatalf("after HEAD moved, content = %q", content)
	}
}

// TestRefreshKnowledge_DeletePrunesOrphanEmbeddings verifies vector rows for
// deleted chunk content are dropped by the embed prune path.
func TestRefreshKnowledge_DeletePrunesOrphanEmbeddings(t *testing.T) {
	t.Parallel()
	r := newKnowledgeTestRepo(t)
	commitFile(t, r.dir, "docs/vec.md", "# Vec\n\nEmbeddable section.\n")
	runGit(t, r.dir, "branch", "-M", "main")
	r.refresh(t)
	_, _, hash, _ := r.chunkRow(t, "docs/vec.md")

	model := "test-model"
	vec := make([]float64, 8)
	for i := range vec {
		vec[i] = float64(i)
	}
	if err := db.StoreKnowledgeEmbeddings(r.indexDB, map[string][]float64{hash: vec}, model); err != nil {
		t.Fatalf("StoreKnowledgeEmbeddings: %v", err)
	}

	runGit(t, r.dir, "rm", "docs/vec.md")
	runGit(t, r.dir, "commit", "-m", "remove vec.md")
	r.refresh(t)
	if err := db.PruneKnowledgeEmbeddings(r.indexDB); err != nil {
		t.Fatalf("PruneKnowledgeEmbeddings: %v", err)
	}
	got, err := db.QueryKnowledgeEmbeddingsByHashes(r.indexDB, model, []string{hash})
	if err != nil {
		t.Fatalf("QueryKnowledgeEmbeddingsByHashes: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("orphan embedding still present for hash %s", hash)
	}
}

// TestIndex_KnowledgeAlwaysBuiltFresh verifies `rekal index` builds knowledge
// into a brand-new index file from current HEAD — stale live knowledge is
// discarded, not patched.
func TestIndex_KnowledgeAlwaysBuiltFresh(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "dev@example.com")
	runGit(t, dir, "config", "user.name", "Dev")
	runGit(t, dir, "config", "gc.auto", "0")
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatalf("mkdir .rekal: %v", err)
	}

	dataDB, err := db.OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := db.InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}
	dataDB.Close()

	commitFile(t, dir, "docs/guide.md", "# Guide\n\nAlphaversion uniquemarker.\n")
	runGit(t, dir, "branch", "-M", "main")

	indexDB, err := db.OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	if err := db.LoadFTSExtension(indexDB); err != nil {
		indexDB.Close()
		t.Skipf("FTS extension unavailable: %v", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		indexDB.Close()
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := refreshKnowledge(nil, indexDB, dir); err != nil {
		indexDB.Close()
		t.Fatalf("initial refresh: %v", err)
	}
	indexDB.Close()

	// HEAD advances; live index is left stale on purpose (no refresh).
	commitFile(t, dir, "docs/guide.md", "# Guide\n\nBetaversion uniquemarker.\n")

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := runIndex(cmd, dir); err != nil {
		t.Fatalf("runIndex: %v", err)
	}
	if _, err := os.Stat(db.IndexPath(dir) + ".rebuilding"); err == nil {
		t.Fatal("leftover index.db.rebuilding after successful rebuild")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat rebuilding: %v", err)
	}

	live, err := db.OpenIndex(dir)
	if err != nil {
		t.Fatalf("reopen index: %v", err)
	}
	defer live.Close()
	if err := db.LoadFTSExtension(live); err != nil {
		t.Fatalf("LoadFTSExtension: %v", err)
	}

	var content string
	if err := live.QueryRow(
		`SELECT content FROM knowledge_chunks WHERE path = $1`, "docs/guide.md").
		Scan(&content); err != nil {
		t.Fatalf("read knowledge after reindex: %v", err)
	}
	if strings.Contains(content, "Alphaversion") {
		t.Fatal("reindex kept stale alpha knowledge from the previous live DB")
	}
	if !strings.Contains(content, "Betaversion") {
		t.Fatalf("reindex knowledge = %q, want Betaversion from HEAD", content)
	}
	wm, ok, err := db.ReadIndexState(live, db.KnowledgeHeadKey)
	if err != nil || !ok {
		t.Fatalf("watermark after reindex: ok=%v err=%v", ok, err)
	}
	if wm != gitx.HeadSHA(dir) {
		t.Fatalf("watermark = %s, want HEAD %s", wm, gitx.HeadSHA(dir))
	}
}
