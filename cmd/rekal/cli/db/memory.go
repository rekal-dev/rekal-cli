package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

const memoryDDL = `CREATE TABLE IF NOT EXISTS memory_revisions (
 id VARCHAR PRIMARY KEY, checkpoint_id VARCHAR NOT NULL,
 source_id VARCHAR NOT NULL, path VARCHAR NOT NULL,
 observed_at VARCHAR NOT NULL, shared BOOLEAN NOT NULL, deleted BOOLEAN NOT NULL,
 content VARCHAR NOT NULL, payload VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS memory_checkpoint ON memory_revisions(checkpoint_id);
CREATE INDEX IF NOT EXISTS memory_source ON memory_revisions(source_id, path);`

func EnsureMemorySchema(d *sql.DB) error { _, err := d.Exec(memoryDDL); return err }

// QueryMemoryRevisions is the archived revision read path, also used by export.
func QueryMemoryRevisions(d *sql.DB, checkpoint string) ([]memory.Revision, error) {
	rows, err := d.Query(`SELECT payload FROM memory_revisions WHERE checkpoint_id = ? ORDER BY source_id, path, id`, checkpoint)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var out []memory.Revision
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var r memory.Revision
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			return nil, err
		}
		if err := r.Validate(); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func LatestMemory(d *sql.DB, source string) (map[string]memory.Revision, error) {
	rows, err := d.Query(`SELECT payload FROM memory_revisions WHERE source_id = ? QUALIFY row_number() OVER (PARTITION BY path ORDER BY CAST(observed_at AS TIMESTAMP_NS) DESC, id DESC) = 1`, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	out := map[string]memory.Revision{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		var r memory.Revision
		if err := json.Unmarshal([]byte(p), &r); err != nil {
			return nil, err
		}
		if err := r.Validate(); err != nil {
			return nil, err
		}
		out[r.Path] = r
	}
	return out, rows.Err()
}

// MemoryChanges compares a complete source scan. Calling it on a failed scan is
// forbidden: absence of a source is not evidence that its notes were deleted.
func MemoryChanges(previous map[string]memory.Revision, source memory.Source, files []memory.File, checkpoint, gitSHA, observed string) []memory.Revision {
	var out []memory.Revision
	seen := map[string]bool{}
	add := func(f memory.File, deleted bool) {
		r := memory.Revision{Schema: 1, CheckpointID: checkpoint, GitSHA: gitSHA, ObservedAt: observed, Source: source, Path: f.Path, Content: f.Content, ModifiedAt: f.ModifiedAt, SourceModifiedAt: f.SourceModifiedAt, NativeType: f.NativeType, Deleted: deleted}
		r.ID = r.Digest()
		out = append(out, r)
	}
	for _, f := range files {
		seen[f.Path] = true
		old, ok := previous[f.Path]
		if ok && !old.Deleted && old.Content == f.Content {
			continue
		}
		add(f, false)
	}
	for p, old := range previous {
		if !seen[p] && !old.Deleted {
			add(memory.File{Path: p}, true)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// InsertMemoryRevisions is atomic for a source scan and idempotent for imports.
// Content-addressed IDs turn a conflicting record into a validation error.
func InsertMemoryRevisions(d *sql.DB, revisions []memory.Revision) error {
	if len(revisions) == 0 {
		return nil
	}
	for _, r := range revisions {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, r := range revisions {
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO memory_revisions VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, r.ID, r.CheckpointID, r.Source.ID(), r.Path, r.ObservedAt, r.Source.Shared, r.Deleted, r.Content, string(b)); err != nil {
			return fmt.Errorf("insert memory: %w", err)
		}
	}
	// Older binaries cannot safely re-export a ledger with memory records.
	if _, err = tx.Exec(`INSERT INTO schema_meta(key,value) VALUES ('schema_version','2') ON CONFLICT(key) DO UPDATE SET value='2'`); err != nil {
		return err
	}
	return tx.Commit()
}

// EnsureMemoryIndex creates disposable archive and current-view tables. The
// archive retains exact historical revisions for the existing query --index path.
func EnsureMemoryIndex(d *sql.DB) error {
	_, err := d.Exec(`CREATE TABLE IF NOT EXISTS memory_archive (
 id VARCHAR PRIMARY KEY, checkpoint_id VARCHAR, source_id VARCHAR, path VARCHAR,
 observed_at VARCHAR, shared BOOLEAN, deleted BOOLEAN, content VARCHAR, payload VARCHAR);
 CREATE TABLE IF NOT EXISTS memory_documents (
 id VARCHAR PRIMARY KEY, source_id VARCHAR, path VARCHAR, content VARCHAR, payload VARCHAR);`)
	return err
}

func refreshMemoryDocuments(d *sql.DB) error {
	_, err := d.Exec(`CREATE OR REPLACE TEMP VIEW memory_current AS
 SELECT id, source_id, path, content, payload, deleted FROM memory_archive
 QUALIFY row_number() OVER (PARTITION BY source_id,path ORDER BY CAST(observed_at AS TIMESTAMP_NS) DESC,id DESC)=1;
 DELETE FROM memory_documents WHERE id NOT IN (SELECT id FROM memory_current WHERE NOT deleted);
 INSERT INTO memory_documents SELECT id,source_id,path,content,payload FROM memory_current c
 WHERE NOT deleted AND NOT EXISTS (SELECT 1 FROM memory_documents d WHERE d.id=c.id);
 DROP VIEW memory_current;`)
	return err
}

// RefreshMemoryIndex runs while data_db is attached. Unchanged rows remain
// untouched; a full rebuild uses exactly the same deterministic projection.
func RefreshMemoryIndex(d *sql.DB) error {
	if err := EnsureMemoryIndex(d); err != nil {
		return err
	}
	if _, err := d.Exec(`INSERT INTO memory_archive SELECT * FROM data_db.memory_revisions ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	return refreshMemoryDocuments(d)
}

func ImportMemoryIndex(d *sql.DB, revisions []memory.Revision) error {
	if err := EnsureMemoryIndex(d); err != nil {
		return err
	}
	for _, r := range revisions {
		if err := r.Validate(); err != nil {
			return err
		}
		if !r.Source.Shared {
			return fmt.Errorf("refuse private memory import")
		}
	}
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, r := range revisions {
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO memory_archive VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, r.ID, r.CheckpointID, r.Source.ID(), r.Path, r.ObservedAt, r.Source.Shared, r.Deleted, r.Content, string(b)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return refreshMemoryDocuments(d)
}
