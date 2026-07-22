package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

// setupDrilldownRepo creates a temp git root with initialized (empty) data
// and index DBs and returns the root path.
func setupDrilldownRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()
	if err := db.InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	indexDB, err := db.OpenIndex(root)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()
	if err := db.InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}

	return root
}

// TestSessionDrilldown_RemoteSessionFromIndex covers the teammate-session
// path: a session that exists only in the index DB (as after `rekal sync`)
// must still be drillable via --session.
func TestSessionDrilldown_RemoteSessionFromIndex(t *testing.T) {
	t.Parallel()

	root := setupDrilldownRepo(t)
	const sessionID = "01REMOTESESSION"

	// Populate index DB only — data.db stays empty (owner-only by design).
	indexDB, err := db.OpenIndex(root)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close()

	if _, err := indexDB.Exec(
		`INSERT INTO session_facets (session_id, user_email, git_branch, actor_type, agent_id, captured_at, turn_count)
		 VALUES ($1, 'greg@example.com', 'feature/sync', 'human', 'claude-code', '2026-06-30 10:00:00', 2)`,
		sessionID,
	); err != nil {
		t.Fatalf("insert session_facets: %v", err)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts) VALUES
		 ($1 || ':0', $1, 0, 'human', 'how does the sync protocol work?', '2026-06-30T10:00:00Z'),
		 ($1 || ':1', $1, 1, 'assistant', 'sync pulls teammate refs into the index db', '2026-06-30T10:00:05Z')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert turns_ft: %v", err)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO tool_calls_index (id, session_id, call_order, tool, path) VALUES
		 ($1 || ':tc0', $1, 0, 'Read', 'cmd/rekal/cli/sync.go')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert tool_calls_index: %v", err)
	}
	if _, err := indexDB.Exec(
		`INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type) VALUES
		 ('cp1', $1, 'cmd/rekal/cli/sync.go', 'modified')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert files_index: %v", err)
	}

	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)

	if err := runSessionDrilldown(cmd, root, sessionID, true, 0, 0, "", false); err != nil {
		t.Fatalf("runSessionDrilldown (remote session): %v", err)
	}

	var got sessionOutput
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %s", err, out.String())
	}

	if got.SessionID != sessionID {
		t.Errorf("session_id = %q, want %q", got.SessionID, sessionID)
	}
	if got.Sid != "s1" {
		t.Errorf("sid = %q, want s1", got.Sid)
	}
	if got.Author != "greg@example.com" {
		t.Errorf("author = %q, want greg@example.com", got.Author)
	}
	if got.Branch != "feature/sync" {
		t.Errorf("branch = %q, want feature/sync", got.Branch)
	}
	if got.TotalTurns != 2 || len(got.Turns) != 2 {
		t.Fatalf("turns = %d (total %d), want 2", len(got.Turns), got.TotalTurns)
	}
	if got.Turns[0].Role != "human" || !strings.Contains(got.Turns[0].Content, "sync protocol") {
		t.Errorf("unexpected first turn: %+v", got.Turns[0])
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Tool != "Read" {
		t.Errorf("unexpected tool calls: %+v", got.ToolCalls)
	}
	if len(got.Files) != 1 || got.Files[0] != "cmd/rekal/cli/sync.go" {
		t.Errorf("unexpected files: %+v", got.Files)
	}

	// --json escape hatch: compact single-line JSON, same payload as the
	// pretty default (the machine-consumer contract that must stay stable
	// across the coming text-default — docs/design/skill-into-command.md §2.0).
	out.Reset()
	if err := runSessionDrilldown(cmd, root, sessionID, true, 0, 0, "", true); err != nil {
		t.Fatalf("runSessionDrilldown (--json): %v", err)
	}
	compact := strings.TrimRight(out.String(), "\n")
	if strings.Contains(compact, "\n") {
		t.Errorf("--json output must be single-line, got:\n%s", compact)
	}
	var viaJSON sessionOutput
	if err := json.Unmarshal([]byte(compact), &viaJSON); err != nil {
		t.Fatalf("--json output must be valid JSON: %v", err)
	}
	if viaJSON.SessionID != sessionID || viaJSON.TotalTurns != got.TotalTurns {
		t.Errorf("--json payload differs from default: %+v vs %+v", viaJSON, got)
	}

	// Short handle must resolve to the same session.
	out.Reset()
	if err := runSessionDrilldown(cmd, root, "s1", false, 0, 0, "", false); err != nil {
		t.Fatalf("runSessionDrilldown (short handle): %v", err)
	}
	var viaShort sessionOutput
	if err := json.Unmarshal([]byte(out.String()), &viaShort); err != nil {
		t.Fatalf("unmarshal short-handle output: %v", err)
	}
	if viaShort.SessionID != sessionID || viaShort.Sid != "s1" || len(viaShort.Turns) != 2 {
		t.Errorf("short-handle drill = %+v, want session %s sid=s1 with 2 turns", viaShort, sessionID)
	}
}

// TestSessionDrilldown_DataSessionStillWorks guards the original path: a
// session in the data DB is served from there.
func TestSessionDrilldown_DataSessionStillWorks(t *testing.T) {
	t.Parallel()

	root := setupDrilldownRepo(t)
	const sessionID = "01LOCALSESSION"

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	if err := db.InsertSession(dataDB, sessionID, "", "hash1", "human", "claude-code", "frank@example.com", "main", "2026-06-30 09:00:00", "claude"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content) VALUES
		 ($1 || ':0', $1, 0, 'human', 'local question')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert turns: %v", err)
	}

	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)

	if err := runSessionDrilldown(cmd, root, sessionID, false, 0, 0, "", false); err != nil {
		t.Fatalf("runSessionDrilldown (local session): %v", err)
	}

	var got sessionOutput
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.SessionID != sessionID || got.Author != "frank@example.com" || len(got.Turns) != 1 {
		t.Errorf("unexpected output: %+v", got)
	}
}

// TestSessionDrilldown_MetadataOmittedForPlainSessions guards cross-agent
// consistency: a session without harness metadata (Codex, old captures, …)
// must not gain the optional keys in its JSON output at all.
func TestSessionDrilldown_MetadataOmittedForPlainSessions(t *testing.T) {
	t.Parallel()

	root := setupDrilldownRepo(t)
	const sessionID = "01CODEXSESSION"

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	// Legacy insert path, codex source, no team/workflow/parent/agent.
	if err := db.InsertSession(dataDB, sessionID, "", "hashx", "human", "", "frank@example.com", "main", "2026-06-30 09:00:00", "codex"); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content) VALUES
		 ($1 || ':0', $1, 0, 'human', 'codex question')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert turns: %v", err)
	}

	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)
	if err := runSessionDrilldown(cmd, root, sessionID, false, 0, 0, "", false); err != nil {
		t.Fatalf("runSessionDrilldown: %v", err)
	}

	raw := out.String()
	for _, key := range []string{"team_name", "workflow_name", "parent_session_id", "agent_id"} {
		if strings.Contains(raw, key) {
			t.Errorf("output contains %q, want it omitted for a session without harness metadata:\n%s", key, raw)
		}
	}
}

// TestSessionDrilldown_SubagentMetadataShown checks that harness metadata is
// carried in the drill-down output when present.
func TestSessionDrilldown_SubagentMetadataShown(t *testing.T) {
	t.Parallel()

	root := setupDrilldownRepo(t)
	const parentID = "01TRUNK"
	const sessionID = "01WORKFLOWSTEP"

	dataDB, err := db.OpenData(root)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close()

	if err := db.InsertSession(dataDB, parentID, "", "hashp", "human", "", "frank@example.com", "main", "2026-06-30 09:00:00", "claude"); err != nil {
		t.Fatalf("InsertSession parent: %v", err)
	}
	if err := db.InsertSessionMeta(dataDB, sessionID, parentID, "hashw", "agent", "step1",
		"frank@example.com", "main", "2026-06-30 09:05:00", "claude", db.SessionMetaFields{TeamName: "perf-team", WorkflowName: "release-flow"}); err != nil {
		t.Fatalf("InsertSessionMeta: %v", err)
	}
	if _, err := dataDB.Exec(
		`INSERT INTO turns (id, session_id, turn_index, role, content) VALUES
		 ($1 || ':0', $1, 0, 'assistant', 'step output')`,
		sessionID,
	); err != nil {
		t.Fatalf("insert turns: %v", err)
	}

	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)
	if err := runSessionDrilldown(cmd, root, sessionID, false, 0, 0, "", false); err != nil {
		t.Fatalf("runSessionDrilldown: %v", err)
	}

	var got sessionOutput
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.TeamName != "perf-team" || got.WorkflowName != "release-flow" ||
		got.ParentSessionID != parentID || got.AgentID != "step1" {
		t.Errorf("metadata = %q/%q/%q/%q, want perf-team/release-flow/%s/step1",
			got.TeamName, got.WorkflowName, got.ParentSessionID, got.AgentID, parentID)
	}
}

// TestSessionDrilldown_NotFoundAnywhere ensures a clear error when the
// session is in neither DB.
func TestSessionDrilldown_NotFoundAnywhere(t *testing.T) {
	t.Parallel()

	root := setupDrilldownRepo(t)

	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)

	err := runSessionDrilldown(cmd, root, "01DOESNOTEXIST", false, 0, 0, "", false)
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
	if !strings.Contains(err.Error(), "session not found") {
		t.Errorf("error = %q, want it to contain 'session not found'", err)
	}
}
