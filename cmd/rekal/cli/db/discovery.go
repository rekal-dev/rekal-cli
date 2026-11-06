package db

import (
	"database/sql"
	"fmt"
)

// DiscoveryRecord is a local accelerator like checkpoint_state. It is never
// serialized to the wire and holds no conversation text.
type DiscoveryRecord struct {
	Adapter, Path, RepoPaths string
	Size, MtimeNS            int64
}

const discoveryCacheDDL = `CREATE TABLE IF NOT EXISTS session_discovery_cache (
	adapter VARCHAR NOT NULL,
	file_path VARCHAR NOT NULL,
	byte_size BIGINT NOT NULL,
	mtime_ns BIGINT NOT NULL,
	repo_paths VARCHAR NOT NULL,
	PRIMARY KEY (adapter, file_path)
);`

func LoadDiscoveryCache(d *sql.DB) ([]DiscoveryRecord, error) {
	rows, err := d.Query(`SELECT adapter, file_path, byte_size, mtime_ns, repo_paths FROM session_discovery_cache`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var records []DiscoveryRecord
	for rows.Next() {
		var r DiscoveryRecord
		if err := rows.Scan(&r.Adapter, &r.Path, &r.Size, &r.MtimeNS, &r.RepoPaths); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func SaveDiscoveryCache(d *sql.DB, records []DiscoveryRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.Prepare(`INSERT INTO session_discovery_cache
		(adapter, file_path, byte_size, mtime_ns, repo_paths) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (adapter, file_path) DO UPDATE SET
		byte_size=excluded.byte_size, mtime_ns=excluded.mtime_ns, repo_paths=excluded.repo_paths`)
	if err != nil {
		return err
	}
	defer stmt.Close() //nolint:errcheck
	for _, r := range records {
		if _, err := stmt.Exec(r.Adapter, r.Path, r.Size, r.MtimeNS, r.RepoPaths); err != nil {
			return fmt.Errorf("cache discovery for %s: %w", r.Path, err)
		}
	}
	return tx.Commit()
}
