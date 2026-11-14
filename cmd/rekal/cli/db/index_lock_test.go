//go:build unix

package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReplaceIndexWaitsForLiveWriter(t *testing.T) {
	live := filepath.Join(t.TempDir(), "index.db")
	rebuilt := live + ".rebuilding"
	old, err := OpenIndexAt(live)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if _, err := old.Exec("CREATE TABLE generation (id INTEGER); INSERT INTO generation VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	fresh, err := OpenIndexAt(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fresh.Exec("CREATE TABLE generation (id INTEGER); INSERT INTO generation VALUES (2)"); err != nil {
		t.Fatal(err)
	}
	if err := fresh.Close(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- ReplaceIndex(rebuilt, live) }()
	select {
	case err := <-done:
		t.Fatalf("replacement bypassed live writer: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	// This write must remain attached to the old database, never the new WAL.
	if _, err := old.Exec("INSERT INTO generation VALUES (3)"); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("replacement did not resume after writer closed")
	}
	got, err := OpenIndexAt(live)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	var count, id int
	if err := got.QueryRow("SELECT count(*), min(id) FROM generation").Scan(&count, &id); err != nil {
		t.Fatal(err)
	}
	if count != 1 || id != 2 {
		t.Fatalf("mixed index generations: count=%d id=%d", count, id)
	}
}

func TestIndexOpenFailureReleasesReplacementLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	if _, err := openCoordinated(path+"?invalid_config_option=1", path); err == nil {
		t.Fatal("expected invalid option error")
	}
	unlock, err := lockIndexPath(path, true)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}
