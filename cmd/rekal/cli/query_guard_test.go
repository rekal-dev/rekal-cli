package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

func TestHasTrailingStatement(t *testing.T) {
	t.Parallel()
	cases := []struct {
		query string
		want  bool
	}{
		// Single statements, with and without the optional terminator.
		{"SELECT 1", false},
		{"SELECT 1;", false},
		{"SELECT 1;  \n\t", false},
		{"SELECT 1; -- explained", false},
		{"SELECT 1; /* explained */", false},
		{"SELECT 1;;;", false},
		// A semicolon that is data, not a separator.
		{"SELECT ';'", false},
		{"SELECT 'a;b' FROM turns WHERE content LIKE '%;%'", false},
		{`SELECT "weird;column" FROM turns`, false},
		{"SELECT 'it''s; fine'", false},
		{"SELECT 1 -- ; not a statement", false},
		{"SELECT 1 /* ; not a statement */", false},
		// The real thing: a second statement riding along.
		{"SELECT 1; DELETE FROM turns", true},
		{"SELECT 1; DELETE FROM turns;", true},
		{"SELECT 1;DROP TABLE sessions", true},
		{"SELECT 1; UPDATE sessions SET user_email='x'", true},
		{"SELECT ';'; DELETE FROM turns", true},
		{"SELECT 1; -- comment\nDELETE FROM turns", true},
		{"SELECT 1; /* comment */ DELETE FROM turns", true},
	}
	for _, c := range cases {
		if got := hasTrailingStatement(c.query); got != c.want {
			t.Errorf("hasTrailingStatement(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}

// TestRunQueryRejectsTrailingStatement pins the message an agent sees, so a
// multi-statement string never reaches the driver.
func TestRunQueryRejectsTrailingStatement(t *testing.T) {
	t.Parallel()
	root := setupDrilldownRepo(t)
	cmd := &cobra.Command{}
	cmd.SetOut(&strings.Builder{})

	err := runQuery(cmd, root, "SELECT 1; DELETE FROM turns", false, false)
	if err == nil {
		t.Fatal("expected a trailing statement to be rejected")
	}
	if !strings.Contains(err.Error(), "one statement per query") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestQueryHandleIsReadOnly is the structural half of the guard: even a
// statement that gets past parsing cannot write, because the handle the query
// path holds is read-only. data.db is append-only, and that has to hold
// regardless of what SQL arrives.
func TestQueryHandleIsReadOnly(t *testing.T) {
	t.Parallel()
	root := setupDrilldownRepo(t)

	d, err := db.OpenDataReadOnly(root)
	if err != nil {
		t.Fatalf("OpenDataReadOnly: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec("DELETE FROM turns"); err == nil {
		t.Fatal("DELETE succeeded on a read-only handle")
	}
	if _, err := d.Exec("CREATE TABLE probe (a INTEGER)"); err == nil {
		t.Fatal("CREATE succeeded on a read-only handle")
	}
	var n int
	if err := d.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n); err != nil {
		t.Fatalf("read-only handle cannot read: %v", err)
	}
}

func TestOpenReadOnlyMissingFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := db.OpenDataReadOnly(root); err == nil {
		t.Fatal("expected an error opening a data DB that does not exist")
	}
}
