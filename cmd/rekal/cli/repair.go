package cli

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
	"github.com/spf13/cobra"
)

func newRepairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Quarantine an unreadable DuckDB store and rebuild it",
		Long: `Recover when .rekal/data.db or index.db fails to open with a DuckDB
serialization error (e.g. "Failed to deserialize: field id mismatch").

Moves the broken file(s) aside (keeps them as *.corrupt-<timestamp>),
creates a fresh data.db, imports checkpoints from your local rekal orphan
branch when present (else fetches origin/rekal/<email>), and rebuilds the
index.

This is the fix when git hooks fail with an opaque DuckDB deserialize
error and every push needs --no-verify. Unmerged local sessions that were
never exported are not on the orphan branch — those stay only in the
quarantined file.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}
			return runRepair(gitRoot, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func runRepair(gitRoot string, stdout, stderr io.Writer) error {
	dataPath := filepath.Join(db.StoreDir(gitRoot), "data.db")
	indexPath := db.IndexPath(gitRoot)

	dataBroken := dbFileUnreadable(dataPath)
	indexBroken := dbFileUnreadable(indexPath)

	if !dataBroken && !indexBroken {
		fmt.Fprintln(stdout, "rekal: data.db and index.db open fine — nothing to repair")
		return nil
	}

	if dataBroken {
		dest, err := db.QuarantineDB(dataPath)
		if err != nil {
			return err
		}
		if dest != "" {
			fmt.Fprintf(stderr, "rekal: quarantined %s → %s\n", dataPath, dest)
		}
		// Index is derived; drop it whenever data is replaced.
		if dest, err := db.QuarantineDB(indexPath); err != nil {
			return err
		} else if dest != "" {
			fmt.Fprintf(stderr, "rekal: quarantined %s → %s\n", indexPath, dest)
		}
	} else if indexBroken {
		dest, err := db.QuarantineDB(indexPath)
		if err != nil {
			return err
		}
		if dest != "" {
			fmt.Fprintf(stderr, "rekal: quarantined %s → %s\n", indexPath, dest)
		}
	}

	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("create fresh data.db: %w", err)
	}
	if err := db.InitDataSchema(dataDB); err != nil {
		_ = dataDB.Close()
		return fmt.Errorf("init data schema: %w", err)
	}
	if err := db.MigrateDataSchema(dataDB); err != nil {
		_ = dataDB.Close()
		return fmt.Errorf("migrate data schema: %w", err)
	}

	imported := 0
	if dataBroken {
		n, src, err := importOwnWire(gitRoot, dataDB, stderr)
		if err != nil {
			_ = dataDB.Close()
			return err
		}
		imported = n
		if src != "" {
			fmt.Fprintf(stdout, "rekal: imported %d session(s) from %s\n", n, src)
		} else {
			fmt.Fprintln(stdout, "rekal: no orphan-branch data to import — fresh empty data.db")
		}
	}
	_ = dataDB.Close()

	if err := doRepairIndex(gitRoot, stderr); err != nil {
		return err
	}

	if dataBroken && imported == 0 {
		fmt.Fprintln(stdout, "rekal: repaired store (empty). Capture sessions with `rekal checkpoint`, or push from a machine that still has a good data.db.")
	} else {
		fmt.Fprintln(stdout, "rekal: repair complete")
	}
	return nil
}

// dbFileUnreadable reports whether path exists and cannot be opened as DuckDB.
func dbFileUnreadable(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	d, err := db.OpenIndexAt(path)
	if err != nil {
		return true
	}
	_ = d.Close()
	return false
}

// importOwnWire loads the user's orphan branch into dataDB. Prefers the local
// ref (survives offline / never-pushed exports); falls back to origin/.
func importOwnWire(gitRoot string, dataDB *sql.DB, stderr io.Writer) (int, string, error) {
	branch := gitx.RekalBranchName()

	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err == nil {
		n, err := transport.ImportBranch(gitRoot, dataDB, branch)
		if err != nil {
			return 0, "", fmt.Errorf("import from %s: %w", branch, err)
		}
		return n, branch, nil
	}

	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		return 0, "", nil
	}
	fetch := exec.Command("git", "-C", gitRoot, "fetch", "origin", branch)
	fetch.Stdin = nil
	if out, err := fetch.CombinedOutput(); err != nil {
		// No remote branch yet — empty store is fine.
		fmt.Fprintf(stderr, "rekal: warning: fetch origin/%s skipped: %s\n", branch, strings.TrimSpace(string(out)))
		return 0, "", nil
	}
	remote := "origin/" + branch
	n, err := transport.ImportBranch(gitRoot, dataDB, remote)
	if err != nil {
		return 0, "", fmt.Errorf("import from %s: %w", remote, err)
	}
	return n, remote, nil
}

// doRepairIndex rebuilds index.db from data.db via the same path as `rekal index`.
func doRepairIndex(gitRoot string, w io.Writer) error {
	cmd := &cobra.Command{}
	cmd.SetOut(w)
	cmd.SetErr(w)
	return runIndex(cmd, gitRoot)
}
