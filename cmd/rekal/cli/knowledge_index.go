package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/knowledge"
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

	var rows []db.KnowledgeChunkRow
	for _, path := range rechunk {
		data := gitx.ShowFile(gitRoot, "HEAD", path)
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
