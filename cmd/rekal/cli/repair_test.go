package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// TestRepair_QuarantinesCorruptDataDBAndReopens is the recovery path for the
// pre-push failure "Failed to deserialize: field id mismatch".
func TestRepair_QuarantinesCorruptDataDBAndReopens(t *testing.T) {
	gitRoot := setupExportTestRepo(t)
	dataPath := filepath.Join(db.StoreDir(gitRoot), "data.db")

	// Overwrite a valid DuckDB file with garbage — open must fail.
	if err := os.WriteFile(dataPath, []byte("this is not a duckdb file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := db.OpenData(gitRoot); err == nil {
		t.Fatal("OpenData should fail on corrupt file")
	} else if !db.IsUnreadableStorage(err) && !strings.Contains(err.Error(), "unreadable") && !strings.Contains(err.Error(), "open database") {
		// Driver wording varies; at minimum open must fail so repair runs.
		t.Logf("OpenData error (ok if not tagged unreadable): %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := runRepair(gitRoot, &stdout, &stderr); err != nil {
		t.Fatalf("runRepair: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "quarantined") {
		t.Fatalf("want quarantine notice, stderr=%q stdout=%q", stderr.String(), stdout.String())
	}

	// Fresh data.db must open and accept schema ops.
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatalf("OpenData after repair: %v", err)
	}
	defer dataDB.Close()
	if err := dataDB.Ping(); err != nil {
		t.Fatalf("Ping after repair: %v", err)
	}

	matches, _ := filepath.Glob(dataPath + ".corrupt-*")
	if len(matches) == 0 {
		t.Fatal("expected quarantined data.db.corrupt-* file")
	}
}
