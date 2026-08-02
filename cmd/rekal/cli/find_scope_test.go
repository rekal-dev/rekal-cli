package cli

import (
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

// TestFindIncludesTeammateSessions covers the reason find sweeps the index: a
// synced teammate session never touches data.db, so a data.db sweep answered
// "every mention" with only this machine's share of the ledger.
func TestFindIncludesTeammateSessions(t *testing.T) {
	t.Parallel()
	root := setupDrilldownRepo(t)

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO sessions (id, session_hash, captured_at, actor_type, agent_id, user_email, branch, source)
		 VALUES ('01LOCALSESSION', 'hash-local', '2026-06-30 10:00:00', 'human', 'claude-code', 'me@example.com', 'main', 'claude')`,
	); err != nil {
		dataDB.Close()
		t.Fatalf("insert sessions: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content, ts)
		 VALUES ('local:0', '01LOCALSESSION', 0, 'human', 'the zstd dictionary is preset', '2026-06-30T10:00:00Z')`,
	); err != nil {
		dataDB.Close()
		t.Fatalf("insert turns: %v", err)
	}
	dataDB.Close()

	indexDB, err := db.OpenIndex(root)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts) VALUES
		 ('local:0', '01LOCALSESSION', 0, 'human', 'the zstd dictionary is preset', '2026-06-30T10:00:00Z'),
		 ('remote:0', '01REMOTESESSION', 0, 'assistant', 'we compress frames with zstd on the wire', '2026-06-30T11:00:00Z')`,
	); err != nil {
		indexDB.Close()
		t.Fatalf("insert turns_ft: %v", err)
	}
	indexDB.Close()

	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runFind(cmd, root, "zstd", ""); err != nil {
		t.Fatalf("runFind: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "01REMOTESESSION") {
		t.Errorf("teammate session missing from the sweep:\n%s", got)
	}
	if !strings.Contains(got, "01LOCALSESSION") {
		t.Errorf("local session missing from the sweep:\n%s", got)
	}
	if !strings.Contains(got, "total 2 mentions in 2 sessions") {
		t.Errorf("expected a complete 2-session total, got:\n%s", got)
	}
}

// TestFindFallsBackToDataDB covers a store whose index has not been built yet:
// the sweep still has to return this machine's own capture.
func TestFindFallsBackToDataDB(t *testing.T) {
	t.Parallel()
	root := setupDrilldownRepo(t)

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO sessions (id, session_hash, captured_at, actor_type, agent_id, user_email, branch, source)
		 VALUES ('01LOCALSESSION', 'hash-local', '2026-06-30 10:00:00', 'human', 'claude-code', 'me@example.com', 'main', 'claude')`,
	); err != nil {
		dataDB.Close()
		t.Fatalf("insert sessions: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content, ts)
		 VALUES ('local:0', '01LOCALSESSION', 0, 'human', 'the zstd dictionary is preset', '2026-06-30T10:00:00Z')`,
	); err != nil {
		dataDB.Close()
		t.Fatalf("insert turns: %v", err)
	}
	dataDB.Close()

	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runFind(cmd, root, "zstd", ""); err != nil {
		t.Fatalf("runFind: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "total 1 mentions in 1 sessions") {
		t.Errorf("expected the data.db fallback to find the local mention, got:\n%s", got)
	}
}
