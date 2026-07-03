//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
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

// TestIndex_FailedRebuildLeavesLiveIndexIntact is the regression test for
// atomic index rebuild: a `rekal index` that fails partway through (here,
// PopulateIndex fails because data.db's sessions table went missing) must
// leave the previous, fully-built index.db completely untouched — not a
// half-dropped/half-repopulated one — and must not leave a stray temp file
// behind in .rekal/.
func TestIndex_FailedRebuildLeavesLiveIndexIntact(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	if _, stderr, err := env.RunCLI("index"); err != nil {
		t.Fatalf("initial index should succeed: %v\nstderr: %s", err, stderr)
	}

	baselineSessions, baselineTurns := indexCounts(t, env)
	if baselineSessions == 0 {
		t.Fatal("expected a non-zero baseline session count after the initial index build")
	}

	// Corrupt data.db so a rebuild's PopulateIndex fails partway through:
	// the very first population step reads FROM data_db.turns.
	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	if _, err := dataDB.Exec("DROP TABLE turns"); err != nil {
		t.Fatalf("drop turns table: %v", err)
	}
	dataDB.Close()

	if _, stderr, err := env.RunCLI("index"); err == nil {
		t.Fatalf("expected the rebuild to fail after corrupting data.db, but it succeeded\nstderr: %s", stderr)
	}

	// The live index.db must be exactly as it was before the failed rebuild.
	sessions, turns := indexCounts(t, env)
	if sessions != baselineSessions || turns != baselineTurns {
		t.Errorf("live index.db changed after a failed rebuild: got sessions=%d turns=%d, want sessions=%d turns=%d",
			sessions, turns, baselineSessions, baselineTurns)
	}

	// No leftover temp file from the failed rebuild.
	tmpPath := db.IndexPath(env.RepoDir) + ".rebuilding"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Errorf("expected no leftover temp file at %s after a failed rebuild", tmpPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error stat-ing temp file: %v", err)
	}
}

// indexCounts reads session/turn counts directly from the live index.db.
func indexCounts(t *testing.T, env *TestEnv) (sessions, turns int) {
	t.Helper()
	indexDB, err := db.OpenIndex(env.RepoDir)
	if err != nil {
		t.Fatalf("open index db: %v", err)
	}
	defer indexDB.Close()

	if err := indexDB.QueryRow("SELECT count(*) FROM session_facets").Scan(&sessions); err != nil {
		t.Fatalf("count session_facets: %v", err)
	}
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&turns); err != nil {
		t.Fatalf("count turns_ft: %v", err)
	}
	return sessions, turns
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

// TestIndex_CachesLSAProjection verifies `rekal index` persists the LSA
// query-projection state, and that recall still returns correct results
// using the cached path — the regression test for recall re-fitting the
// whole LSA model (tokenizing every session, re-running SVD) on every
// single query instead of reusing what `rekal index` already computed.
func TestIndex_CachesLSAProjection(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	if _, stderr, err := env.RunCLI("index"); err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr)
	}

	indexDB, err := db.OpenIndex(env.RepoDir)
	if err != nil {
		t.Fatalf("open index db: %v", err)
	}
	defer indexDB.Close()

	var value string
	err = indexDB.QueryRow(`SELECT value FROM index_state WHERE key = 'lsa_projection'`).Scan(&value)
	if err != nil {
		t.Fatalf("expected lsa_projection to be cached in index_state after `rekal index`: %v", err)
	}
	if value == "" {
		t.Error("lsa_projection value is empty")
	}

	// Recall must still work correctly using the cached projection.
	stdout, _, err := env.RunCLI("JWT auth")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}
	results, _ := output["results"].([]interface{})
	if len(results) == 0 {
		t.Error("expected at least one recall result using the cached LSA projection")
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

func TestQuery_SessionPagination(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "test-session-1", "--limit", "2")
	if err != nil {
		t.Fatalf("query --session --limit should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	turns, ok := output["turns"].([]interface{})
	if !ok {
		t.Fatal("expected turns array")
	}
	if len(turns) != 2 {
		t.Errorf("expected 2 turns, got %d", len(turns))
	}

	if total, _ := output["total_turns"].(float64); int(total) != 4 {
		t.Errorf("expected total_turns=4, got %v", output["total_turns"])
	}

	if hasMore, _ := output["has_more"].(bool); !hasMore {
		t.Error("expected has_more=true")
	}
}

func TestQuery_SessionPagination_Offset(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "test-session-1", "--offset", "2", "--limit", "2")
	if err != nil {
		t.Fatalf("query --session --offset --limit should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	turns, ok := output["turns"].([]interface{})
	if !ok {
		t.Fatal("expected turns array")
	}
	if len(turns) != 2 {
		t.Errorf("expected 2 turns, got %d", len(turns))
	}

	if hasMore, _ := output["has_more"].(bool); hasMore {
		t.Error("expected has_more=false")
	}

	// Verify these are the last 2 turns (index 2 and 3).
	first := turns[0].(map[string]interface{})
	if idx, _ := first["index"].(float64); int(idx) != 2 {
		t.Errorf("expected first turn index=2, got %v", first["index"])
	}
}

func TestQuery_SessionRoleFilter(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "test-session-1", "--role", "human")
	if err != nil {
		t.Fatalf("query --session --role should succeed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	turns, ok := output["turns"].([]interface{})
	if !ok {
		t.Fatal("expected turns array")
	}

	// Session 1 has 2 human turns (index 0 and 2).
	if len(turns) != 2 {
		t.Errorf("expected 2 human turns, got %d", len(turns))
	}

	for _, turn := range turns {
		m := turn.(map[string]interface{})
		if m["role"] != "human" {
			t.Errorf("expected role=human, got %v", m["role"])
		}
	}

	// total_turns should reflect the filtered count.
	if total, _ := output["total_turns"].(float64); int(total) != 2 {
		t.Errorf("expected total_turns=2 (human only), got %v", output["total_turns"])
	}
}

func TestQuery_SessionPaginationRequiresSession(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	_, _, err := env.RunCLI("query", "--offset", "1", "SELECT 1")
	if err == nil {
		t.Error("expected error for --offset without --session")
	}
}

func TestRecall_GroupsSubagentUnderTrunk(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedWorkflowData(t, env)

	if _, _, err := env.RunCLI("index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	stdout, _, err := env.RunCLI("release checklist")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}

	var output struct {
		Results []struct {
			SessionID string `json:"session_id"`
			Session   struct {
				WorkflowName string `json:"workflow_name,omitempty"`
			} `json:"session"`
			Children []struct {
				SessionID string `json:"session_id"`
				Session   struct {
					AgentID      string `json:"agent_id,omitempty"`
					WorkflowName string `json:"workflow_name,omitempty"`
				} `json:"session"`
			} `json:"children,omitempty"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	// The trunk session and its two workflow-step transcripts all match
	// "release checklist" — they must collapse into one top-level result,
	// not three.
	var trunkResults int
	for _, r := range output.Results {
		if r.SessionID == "trunk-session" {
			trunkResults++
			if len(r.Children) == 0 {
				t.Errorf("expected trunk result to have folded children, got none")
			}
			for _, c := range r.Children {
				if c.SessionID == "trunk-session" {
					t.Errorf("trunk session should not appear in its own children")
				}
			}
		}
	}
	if trunkResults != 1 {
		t.Errorf("expected exactly 1 top-level result for trunk-session, got %d (results: %+v)", trunkResults, output.Results)
	}
}

func TestRecall_SteeringTurnBoosted(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}

	if err := db.InsertSession(dataDB, "steer-session", "", "hash-steer", "human", "", "carol@example.com", "main", "2026-02-25T12:00:00Z", ""); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-s1", "steer-session", 0, "human", "look into the deploy pipeline", "2026-02-25T12:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-s2", "steer-session", 1, "assistant", "Checking the pipeline config now.", "2026-02-25T12:01:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	// A steering message mentions the search term directly — should win the
	// best-turn slot over the plain human/assistant turns above.
	if err := db.InsertTurn(dataDB, "turn-s3", "steer-session", 2, "human_steering", "actually skip the canary rollout entirely", "2026-02-25T12:02:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	dataDB.Close()

	if _, _, err := env.RunCLI("index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	stdout, _, err := env.RunCLI("canary rollout")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}

	var output struct {
		Results []struct {
			SessionID   string `json:"session_id"`
			SnippetRole string `json:"snippet_role"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if len(output.Results) == 0 {
		t.Fatal("expected at least one result")
	}
	if output.Results[0].SessionID != "steer-session" {
		t.Fatalf("expected steer-session to be top result, got %s", output.Results[0].SessionID)
	}
	if output.Results[0].SnippetRole != "human_steering" {
		t.Errorf("expected snippet_role=human_steering, got %s", output.Results[0].SnippetRole)
	}
}

func TestRecall_PerConversationBudgetCapsChildren(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}

	if err := db.InsertSession(dataDB, "big-trunk", "", "hash-big-trunk", "human", "", "erin@example.com", "main", "2026-02-25T13:00:00Z", ""); err != nil {
		t.Fatalf("insert trunk session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-bt", "big-trunk", 0, "human", "audit every service for the security sweep", "2026-02-25T13:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	// A large dynamic workflow spawns many step transcripts, all matching
	// the query — the budget must cap how many get nested, so this one
	// conversation cannot fill the whole children list unbounded.
	for i := 0; i < 8; i++ {
		sid := fmt.Sprintf("sweep-step-%d", i)
		if err := db.InsertSessionMeta(dataDB, sid, "big-trunk", "hash-"+sid, "agent", fmt.Sprintf("agent-%d", i), "", "main", "2026-02-25T13:0"+fmt.Sprint(i+1)+":00Z", "", db.SessionMetaFields{WorkflowName: "security-sweep"}); err != nil {
			t.Fatalf("insert step %s: %v", sid, err)
		}
		if err := db.InsertTurn(dataDB, "turn-"+sid, sid, 0, "assistant", "security sweep step: scanning service dependencies", "2026-02-25T13:0"+fmt.Sprint(i+1)+":00Z"); err != nil {
			t.Fatalf("insert turn for %s: %v", sid, err)
		}
	}
	dataDB.Close()

	if _, _, err := env.RunCLI("index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	stdout, _, err := env.RunCLI("security sweep")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}

	var output struct {
		Results []struct {
			SessionID string `json:"session_id"`
			Children  []struct {
				SessionID string `json:"session_id"`
			} `json:"children,omitempty"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	for _, r := range output.Results {
		if r.SessionID != "big-trunk" {
			continue
		}
		if len(r.Children) >= 8 {
			t.Errorf("expected children capped below the full 8 matching steps, got %d", len(r.Children))
		}
		return
	}
	t.Fatal("expected a top-level result for big-trunk")
}

func TestQuery_SessionDrilldown_ChildSessionIDs(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	seedWorkflowData(t, env)

	stdout, _, err := env.RunCLI("query", "--session", "trunk-session")
	if err != nil {
		t.Fatalf("query --session should succeed: %v", err)
	}

	var output struct {
		ChildSessionIDs []string `json:"child_session_ids"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("expected valid JSON: %v\nstdout: %s", err, stdout)
	}

	if len(output.ChildSessionIDs) != 2 {
		t.Fatalf("expected 2 child session ids, got %d: %v", len(output.ChildSessionIDs), output.ChildSessionIDs)
	}
}

// seedWorkflowData inserts a trunk session plus two dynamic-workflow-step
// transcripts linked to it via parent_session_id, all discussing the same
// topic so they compete for the same search results.
func seedWorkflowData(t *testing.T, env *TestEnv) {
	t.Helper()

	dataDB, err := db.OpenData(env.RepoDir)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	defer dataDB.Close()

	if err := db.InsertSession(dataDB, "trunk-session", "", "hash-trunk", "human", "", "dave@example.com", "main", "2026-02-25T09:00:00Z", ""); err != nil {
		t.Fatalf("insert trunk session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-t1", "trunk-session", 0, "human", "run the release checklist before we ship", "2026-02-25T09:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-t2", "trunk-session", 1, "assistant", "Kicking off the release checklist workflow.", "2026-02-25T09:01:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	if err := db.InsertSessionMeta(dataDB, "workflow-step-1", "trunk-session", "hash-wf1", "agent", "agent-1", "", "main", "2026-02-25T09:02:00Z", "", db.SessionMetaFields{WorkflowName: "release-checklist"}); err != nil {
		t.Fatalf("insert workflow step 1: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-w1", "workflow-step-1", 0, "assistant", "Running the release checklist: verifying CI is green.", "2026-02-25T09:02:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	if err := db.InsertSessionMeta(dataDB, "workflow-step-2", "trunk-session", "hash-wf2", "agent", "agent-2", "", "main", "2026-02-25T09:03:00Z", "", db.SessionMetaFields{WorkflowName: "release-checklist"}); err != nil {
		t.Fatalf("insert workflow step 2: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-w2", "workflow-step-2", 0, "assistant", "Running the release checklist: tagging the release.", "2026-02-25T09:03:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
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
	if err := db.InsertSession(dataDB, "test-session-1", "", "hash1", "human", "", "alice@example.com", "feature/auth", "2026-02-25T10:00:00Z", ""); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-1", "test-session-1", 0, "human", "fix the JWT expiry bug in the auth middleware", "2026-02-25T10:00:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-2", "test-session-1", 1, "assistant", "Let me read the JWT middleware file to understand the expiry logic.", "2026-02-25T10:01:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-2b", "test-session-1", 2, "human", "Now fix the token refresh endpoint too.", "2026-02-25T10:02:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertTurn(dataDB, "turn-2c", "test-session-1", 3, "assistant", "I'll update the refresh endpoint to use the new expiry configuration.", "2026-02-25T10:03:00Z"); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	if err := db.InsertToolCall(dataDB, "tc-1", "test-session-1", 0, "Read", "src/auth/middleware.go", ""); err != nil {
		t.Fatalf("insert tool_call: %v", err)
	}
	if err := db.InsertToolCall(dataDB, "tc-2", "test-session-1", 1, "Edit", "src/auth/jwt.go", ""); err != nil {
		t.Fatalf("insert tool_call: %v", err)
	}

	// Session 2: DB topic.
	if err := db.InsertSession(dataDB, "test-session-2", "", "hash2", "human", "", "bob@example.com", "feature/db", "2026-02-25T11:00:00Z", ""); err != nil {
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
