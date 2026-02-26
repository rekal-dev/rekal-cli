//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
)

func TestIndex_Rebuild(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Seed data DB with a session and turns.
	seedData(t, env)

	stdout, stderr, err := env.RunCLI("index")
	if err != nil {
		t.Fatalf("index should succeed: %v\nstderr: %s", err, stderr)
	}
	_ = stdout

	if !strings.Contains(stderr, "index rebuilt") {
		t.Errorf("expected 'index rebuilt' in stderr, got: %q", stderr)
	}
}

func TestRecall_HybridSearch(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	// Build index first.
	_, _, err := env.RunCLI("index")
	if err != nil {
		t.Fatalf("index failed: %v", err)
	}

	stdout, _, err := env.RunCLI("JWT auth")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if output["mode"] != "hybrid" {
		t.Errorf("expected mode=hybrid, got %v", output["mode"])
	}
}

func TestRecall_FilterOnly(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	_, _, err := env.RunCLI("index")
	if err != nil {
		t.Fatalf("index failed: %v", err)
	}

	stdout, _, err := env.RunCLI("--actor", "human")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if output["mode"] != "filter" {
		t.Errorf("expected mode=filter, got %v", output["mode"])
	}
}

func TestRecall_AutoRebuild(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	// Don't run index first — recall should auto-rebuild.
	stdout, stderr, err := env.RunCLI("JWT")
	if err != nil {
		t.Fatalf("recall should succeed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stderr, "index not built") {
		t.Errorf("expected auto-rebuild message, got stderr: %q", stderr)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}
}

func TestQuery_SessionDrilldown(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "test-session-1")
	if err != nil {
		t.Fatalf("query --session should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if output["session_id"] != "test-session-1" {
		t.Errorf("expected session_id=test-session-1, got %v", output["session_id"])
	}

	turns, ok := output["turns"].([]interface{})
	if !ok || len(turns) == 0 {
		t.Error("expected non-empty turns array")
	}
}

func TestQuery_SessionDrilldown_Full(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "test-session-1", "--full")
	if err != nil {
		t.Fatalf("query --session --full should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if _, ok := output["tool_calls"]; !ok {
		t.Error("expected tool_calls in full output")
	}
}

func TestQuery_SessionAndSQL_MutuallyExclusive(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	_, _, err := env.RunCLI("query", "--session", "foo", "SELECT 1")
	if err == nil {
		t.Error("expected error for --session + SQL")
	}
}

// seedData inserts test sessions, turns, tool_calls, checkpoints into the data DB.
func seedData(t *testing.T, env *TestEnv) {
	t.Helper()

	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	defer dataDB.Close()

	// Session 1: JWT auth topic.
	if err := db.InsertSession(dataDB, "test-session-1", "", "hash1", "human", "", "alice@example.com", "feature/auth", "2026-02-25T10:00:00Z"); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-1", "test-session-1", 0, "human", "fix the JWT expiry bug in the auth middleware", "2026-02-25T10:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-2", "test-session-1", 1, "assistant", "Let me read the JWT middleware file to understand the expiry logic.", "2026-02-25T10:01:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertToolCall(dataDB, "tc-1", "test-session-1", 0, "Read", "src/auth/middleware.go", ""); err != nil {
		t.Fatalf("insert tool_call: %v", err)
	}
	if err := db.InsertToolCall(dataDB, "tc-2", "test-session-1", 1, "Edit", "src/auth/jwt.go", ""); err != nil {
		t.Fatalf("insert tool_call: %v", err)
	}

	// Session 2: DB topic.
	if err := db.InsertSession(dataDB, "test-session-2", "", "hash2", "human", "", "bob@example.com", "feature/db", "2026-02-25T11:00:00Z"); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-3", "test-session-2", 0, "human", "optimize the database connection pooling", "2026-02-25T11:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-4", "test-session-2", 1, "assistant", "I'll look at the connection pool configuration.", "2026-02-25T11:01:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	// Checkpoint linking session 1.
	if err := db.InsertCheckpoint(dataDB, "cp-1", "abc123", "feature/auth", "alice@example.com", "2026-02-25T10:05:00Z", "human", ""); err != nil {
		t.Fatalf("insert checkpoint: %v", err)
	}
	if err := db.InsertCheckpointSession(dataDB, "cp-1", "test-session-1"); err != nil {
		t.Fatalf("insert checkpoint_session: %v", err)
	}
	if err := db.InsertFileTouched(dataDB, "ft-1", "cp-1", "src/auth/middleware.go", "M"); err != nil {
		t.Fatalf("insert file_touched: %v", err)
	}
	if err := db.InsertFileTouched(dataDB, "ft-2", "cp-1", "src/auth/jwt.go", "M"); err != nil {
		t.Fatalf("insert file_touched: %v", err)
	}

	// Checkpoint linking session 2.
	if err := db.InsertCheckpoint(dataDB, "cp-2", "def456", "feature/db", "bob@example.com", "2026-02-25T11:05:00Z", "human", ""); err != nil {
		t.Fatalf("insert checkpoint: %v", err)
	}
	if err := db.InsertCheckpointSession(dataDB, "cp-2", "test-session-2"); err != nil {
		t.Fatalf("insert checkpoint_session: %v", err)
	}
}
