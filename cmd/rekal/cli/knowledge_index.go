package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/knowledge"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
)

// refreshKnowledge brings the index's knowledge layer (chunked prose files —
// docs/design/knowledge-layer.md) up to date with HEAD. Watermark-gated: when
// the indexed commit SHA matches HEAD it returns immediately, so the steady
// state costs one rev-parse. On a mismatch it diffs stored blob SHAs against
// `git ls-tree -r HEAD` and re-chunks only the files whose content changed —
// git's content addressing is the change detector, no mtime heuristics.
//
// Callers treat this as best-effort: recall must keep working (with a stale
// or absent knowledge layer) when the refresh fails. The FTS extension must
// already be loaded on indexDB.
func refreshKnowledge(w io.Writer, indexDB *sql.DB, gitRoot string) error {
	head := gitx.HeadSHA(gitRoot)
	if strings.Trim(head, "0") == "" {
		return nil // no commits yet — nothing at HEAD to index
	}
	if stored, ok, _ := db.ReadIndexState(indexDB, db.KnowledgeHeadKey); ok && stored == head {
		return nil
	}

	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		return err
	}

	current := proseBlobs(gitRoot)
	indexed, err := db.QueryKnowledgeBlobs(indexDB)
	if err != nil {
		return err
	}

	// Paths to re-chunk (new or changed blob) and paths to drop (deleted, or
	// changed — changed paths are deleted first, then re-inserted).
	var stale, rechunk []string
	for path, sha := range current {
		if indexed[path] != sha {
			rechunk = append(rechunk, path)
			if _, existed := indexed[path]; existed {
				stale = append(stale, path)
			}
		}
	}
	for path := range indexed {
		if _, ok := current[path]; !ok {
			stale = append(stale, path)
		}
	}

	if len(stale) == 0 && len(rechunk) == 0 {
		// Prose corpus unchanged (e.g. code-only commits) — just move the
		// watermark.
		return db.WriteIndexState(indexDB, db.KnowledgeHeadKey, head)
	}

	if err := db.DeleteKnowledgeChunks(indexDB, stale); err != nil {
		return err
	}

	// One `git cat-file --batch` for every blob to (re)chunk — a full build
	// on a docs-heavy repo would otherwise spawn one `git show` per file.
	shas := make([]string, 0, len(rechunk))
	seen := make(map[string]bool, len(rechunk))
	for _, path := range rechunk {
		if sha := current[path]; !seen[sha] {
			seen[sha] = true
			shas = append(shas, sha)
		}
	}
	blobs := gitx.BlobContents(gitRoot, shas)

	var rows []db.KnowledgeChunkRow
	for _, path := range rechunk {
		data := blobs[current[path]]
		if data == nil {
			continue
		}
		for _, c := range knowledge.ChunkFile(path, data) {
			rows = append(rows, db.KnowledgeChunkRow{
				ID:          fmt.Sprintf("%s#%d", path, c.StartLine),
				Path:        path,
				Anchor:      c.Anchor,
				Breadcrumb:  c.Breadcrumb,
				StartLine:   c.StartLine,
				EndLine:     c.EndLine,
				Content:     c.Content,
				ContentHash: c.Hash,
				BlobSHA:     current[path],
			})
		}
	}
	if err := db.InsertKnowledgeChunks(indexDB, rows); err != nil {
		return err
	}

	// Rebuild the FTS index over the updated chunk set (guarded: no chunks,
	// no index — and the search side fails soft without one).
	if err := db.CreateKnowledgeFTSIndex(indexDB); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, db.KnowledgeHeadKey, head); err != nil {
		return err
	}
	if w != nil {
		fmt.Fprintf(w, "knowledge layer: %d file(s) re-chunked, %d chunk(s)\n", len(rechunk), len(rows))
	}
	return nil
}

// knowledgeEmbedBudget caps how many chunks one budgeted embedding pass will
// embed (post-commit hook and each bite of `rekal embed`). A giant prose
// import converges across bites/commits; un-embedded chunks stay
// keyword-findable (the two-speed contract). Kept small (like
// sessionEmbedBudget) to bound how long one bite pins the index write lock, so
// a concurrent recall waits out at most a short bite — see db.open's retry.
const knowledgeEmbedBudget = 16

// embedKnowledgeChunks builds semantic vectors for knowledge chunks that
// don't have one yet, through the same content-hash-keyed embedding cache the
// session layer uses (.rekal/embed-cache.db). budget 0 = embed everything
// missing; budget N = at most N chunks this pass (the missing-vectors join
// drives convergence across passes). Callers treat this as non-fatal — the
// knowledge layer degrades to keyword-only, exactly like the session layer's
// semantic pass.
//
// Not called at recall time: recall latency stays pure read. Vectors are
// built by `rekal embed` (background after index/sync, or by hand) and the
// post-commit hook (budgeted).
//
// If emb is nil this constructs (and closes) an embedder for the call; the
// background embed loop passes a shared, already-loaded one so the model loads
// once, off the index lock (see runEmbed).
func embedKnowledgeChunks(w io.Writer, indexDB *sql.DB, gitRoot string, budget int, emb sessionEmbedder) error {
	model := intendedEmbedModel(gitRoot)
	if model == "" {
		return nil // no semantic backend on this platform/config — keyword-only
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		return err
	}
	// Drop vectors for content that no longer exists in any chunk (deleted
	// or rewritten sections). The embedcache still holds them, so a reverted
	// edit re-fills from cache, not from the model.
	if err := db.PruneKnowledgeEmbeddings(indexDB); err != nil {
		return err
	}
	missing, err := db.QueryKnowledgeChunksMissingEmbeddings(indexDB, model, budget)
	if err != nil || len(missing) == 0 {
		return err
	}

	vectors := make(map[string][]float64, len(missing))
	hashes := make([]string, 0, len(missing))
	for h := range missing {
		hashes = append(hashes, h)
	}
	cacheDB, cacheErr := db.OpenEmbedCache(gitRoot)
	if cacheErr == nil {
		defer cacheDB.Close() //nolint:errcheck
		if got, gerr := db.CacheGetEmbeddings(cacheDB, model, hashes); gerr == nil {
			for h, v := range got {
				vectors[h] = v
				delete(missing, h)
			}
		}
	}

	embedded := 0
	if len(missing) > 0 {
		if emb == nil {
			var cerr error
			emb, cerr = semanticEmbedder(gitRoot)
			if cerr != nil || emb == nil {
				// Backend unavailable — still store whatever the cache supplied.
				if serr := db.StoreKnowledgeEmbeddings(indexDB, vectors, model); serr != nil {
					return serr
				}
				return cerr
			}
			defer emb.Close()
		}
		fresh, err := emb.EmbedSessions(missing) // keys are content hashes
		if err != nil {
			if serr := db.StoreKnowledgeEmbeddings(indexDB, vectors, model); serr != nil {
				return serr
			}
			return err
		}
		for h, v := range fresh {
			vectors[h] = v
		}
		embedded = len(fresh)
		if cacheErr == nil {
			_ = db.CachePutEmbeddings(cacheDB, model, fresh) // best-effort
		}
	}

	if err := db.StoreKnowledgeEmbeddings(indexDB, vectors, model); err != nil {
		return err
	}
	if w != nil && len(vectors) > 0 {
		fmt.Fprintf(w, "knowledge embeddings: %d stored (%d cached, %d embedded)\n",
			len(vectors), len(vectors)-embedded, embedded)
	}
	return nil
}

// intendedEmbedModel names the semantic model this repo's config would embed
// with, without instantiating a backend — the embedded nomic client load is
// far too heavy for the cheap "is anything missing?" gate.
func intendedEmbedModel(gitRoot string) string {
	cfg, err := readMergedConfig(gitRoot)
	if err == nil && cfg.Embedding != nil {
		if ec, eerr := cfg.Embedding.resolve(); eerr == nil {
			return ec.Model
		}
	}
	if nomic.Supported() {
		return nomic.ModelName
	}
	return ""
}

// proseBlobs filters the tracked-files listing at HEAD down to knowledge
// material (markdown and plain text). Tracked-only by construction —
// gitignored/vendored/generated files never appear, and indexing adds zero
// wire cost because every indexed byte is already in git.
func proseBlobs(gitRoot string) map[string]string {
	all := gitx.TrackedBlobs(gitRoot, "HEAD")
	prose := make(map[string]string, len(all)/10+1)
	for path, sha := range all {
		if knowledge.IsProseFile(path) {
			prose[path] = sha
		}
	}
	return prose
}
