package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	duckdb "github.com/marcboeker/go-duckdb"
)

// The lock is attached to the pathname, not the replaceable database inode.
// Keep it until database/sql closes its connector (after all connections).
// DuckDB's own inode lock cannot protect a rename from a live background writer:
// that writer would keep writing index.db.wal beside a different database.
type indexConnector struct {
	driver.Connector
	closeDB func() error
	unlock  func()
	once    sync.Once
	err     error
}

func (c *indexConnector) Close() error {
	c.once.Do(func() {
		c.err = c.closeDB()
		c.unlock()
	})
	return c.err
}

func openCoordinated(dsn, path string) (*sql.DB, error) {
	if filepath.Base(path) != "index.db" {
		return sql.Open("duckdb", dsn)
	}
	unlock, err := lockIndexPath(path, false)
	if err != nil {
		return nil, err
	}
	connector, err := duckdb.NewConnector(dsn, nil)
	if err != nil {
		unlock()
		return nil, err
	}
	return sql.OpenDB(&indexConnector{Connector: connector, closeDB: connector.Close, unlock: unlock}), nil
}

// ReplaceIndex waits for all live index handles to close before publishing a
// rebuilt file. A WAL belongs to its original database and must never be replayed
// against the replacement. On failure the live database is left in place.
func ReplaceIndex(rebuilt, live string) error {
	unlock, err := lockIndexPath(live, true)
	if err != nil {
		return err
	}
	defer unlock()
	// A crashed writer may have left a WAL. Recover/checkpoint it against the OLD
	// database before replacing it; never delete uncheckpointed history blindly.
	if _, err := os.Stat(live + ".wal"); err == nil {
		old, err := sql.Open("duckdb", live)
		if err != nil {
			return fmt.Errorf("open old index before replacement: %w", err)
		}
		_, checkpointErr := old.Exec("CHECKPOINT")
		closeErr := old.Close()
		if checkpointErr != nil {
			return fmt.Errorf("checkpoint old index: %w", checkpointErr)
		}
		if closeErr != nil {
			return closeErr
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(live + ".wal"); err == nil {
		return fmt.Errorf("old index WAL remains; refusing replacement")
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(rebuilt, live)
}
