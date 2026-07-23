//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var ulidRE = regexp.MustCompile(`01[0-9A-HJKMNP-TV-Z]{24}`)

// TestRecallGraph_L1_CaptureDrainHint exercises the whole L1 loop: a drill
// spools an edge, the next checkpoint drains it into data.db.recall_edges and
// the derived session_reach aggregate, and a later recall surfaces the reach
// hint on the previously-reached session — while a never-reached session shows
// none.
func TestRecallGraph_L1_CaptureDrainHint(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// --- Capture session 1 (the future reach target). ---
	cleanup1 := writeSessionFile(t, env.RepoDir, "session1.jsonl", testSessionJSONL)
	defer cleanup1()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "fix auth bug")
	if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 1: %v (%s)", err, stderr)
	}

	// Resolve S1's ULID.
	idsOut, _, err := env.RunCLI("query", "--json", "SELECT id FROM sessions")
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	s1 := ulidRE.FindString(idsOut)
	if s1 == "" {
		t.Fatalf("no session id in %q", idsOut)
	}

	// --- Drill S1: the strong "this memory was used" signal → spool a drill edge. ---
	if _, stderr, err := env.RunCLI("query", "--session", s1, "--limit", "1"); err != nil {
		t.Fatalf("drill S1: %v (%s)", err, stderr)
	}
	// The spool exists before the drain.
	if _, err := os.Stat(filepath.Join(env.RepoDir, ".rekal", "recall-log.ndjson")); err != nil {
		t.Fatalf("expected recall spool after drill: %v", err)
	}

	// --- Capture session 2: this checkpoint drains the spool into data.db. ---
	cleanup2 := writeSessionFile(t, env.RepoDir, "session2.jsonl", testSessionJSONL2)
	defer cleanup2()
	if err := os.WriteFile(filepath.Join(env.RepoDir, "login.go"), []byte("func login() error { log.Println(\"ok\"); return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "add logging")
	if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 2: %v (%s)", err, stderr)
	}

	// The drill edge is now the permanent record, and the spool is gone.
	assertQueryContains(t, env, "SELECT count(*) as n FROM recall_edges", `"n":1`)
	assertQueryContains(t, env, "SELECT kind FROM recall_edges", `"kind":"drill"`)
	assertQueryContains(t, env, "SELECT target_session_id as tgt FROM recall_edges", `"tgt":"`+s1+`"`)
	if _, err := os.Stat(filepath.Join(env.RepoDir, ".rekal", "recall-log.ndjson")); !os.IsNotExist(err) {
		t.Errorf("spool should be drained (removed) after checkpoint, stat err=%v", err)
	}

	// The derived aggregate has S1.
	reachOut, _, err := env.RunCLI("query", "--index", "--json", "SELECT target_session_id as t, reach_count as c FROM session_reach")
	if err != nil {
		t.Fatalf("query session_reach: %v", err)
	}
	if !strings.Contains(reachOut, `"c":1`) || !strings.Contains(reachOut, s1) {
		t.Errorf("session_reach missing S1 count 1: %q", reachOut)
	}

	// --- Recall surfaces the reach hint on S1, and nothing on the never-reached S2. ---
	recallOut, _, err := env.RunCLI("fix the auth bug in login", "--json")
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	var payload struct {
		Results []struct {
			SessionID string `json:"session_id"`
			Reached   *struct {
				Count int    `json:"count"`
				Query string `json:"query"`
			} `json:"reached"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(recallOut), &payload); err != nil {
		t.Fatalf("unmarshal recall JSON: %v\n%s", err, recallOut)
	}
	var sawS1 bool
	for _, r := range payload.Results {
		switch r.SessionID {
		case s1:
			sawS1 = true
			if r.Reached == nil || r.Reached.Count != 1 {
				t.Errorf("S1 reached hint = %+v, want count 1 (from the drill; the current recall must not self-inflate it)", r.Reached)
			}
		default:
			// Any other surfaced session was never reached before → no hint.
			if r.Reached != nil {
				t.Errorf("session %s has an unexpected reach hint %+v", r.SessionID, r.Reached)
			}
		}
	}
	if !sawS1 {
		t.Fatalf("recall did not surface S1; results: %s", recallOut)
	}
}

// TestRecallGraph_L1_DrainsWithoutNewSession guards the drain-gap fix: an agent
// that recalls/drills inside an already-checkpointed session, then checkpoints
// on a commit that captures no new session, must still have its spooled edges
// drained into recall_edges and session_reach refreshed — the graph must not
// stall waiting for a fresh session.
func TestRecallGraph_L1_DrainsWithoutNewSession(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	if err := os.WriteFile(filepath.Join(env.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, env.RepoDir, "initial")

	// Capture one session.
	cleanup := writeSessionFile(t, env.RepoDir, "s.jsonl", testSessionJSONL)
	defer cleanup()
	gitCommit(t, env.RepoDir, "work")
	if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 1: %v (%s)", err, stderr)
	}
	idsOut, _, err := env.RunCLI("query", "--json", "SELECT id FROM sessions")
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	s1 := ulidRE.FindString(idsOut)
	if s1 == "" {
		t.Fatalf("no session id in %q", idsOut)
	}

	// Drill it → spool an edge.
	if _, _, err := env.RunCLI("query", "--session", s1, "--limit", "1"); err != nil {
		t.Fatalf("drill: %v", err)
	}

	// Checkpoint on a commit that captures NO new session (same transcript,
	// empty commit). Before the fix this returned early and never drained.
	gitCommit(t, env.RepoDir, "empty follow-up")
	if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint 2 (no new session): %v (%s)", err, stderr)
	}

	// The drill edge must have landed and the aggregate refreshed.
	assertQueryContains(t, env, "SELECT count(*) as n FROM recall_edges", `"n":1`)
	reachOut, _, err := env.RunCLI("query", "--index", "--json", "SELECT target_session_id as t, reach_count as c FROM session_reach")
	if err != nil {
		t.Fatalf("query session_reach: %v", err)
	}
	if !strings.Contains(reachOut, s1) || !strings.Contains(reachOut, `"c":1`) {
		t.Errorf("session_reach not refreshed after no-new-session checkpoint: %q", reachOut)
	}
	// Spool is drained (removed).
	if _, err := os.Stat(filepath.Join(env.RepoDir, ".rekal", "recall-log.ndjson")); !os.IsNotExist(err) {
		t.Errorf("spool should be drained, stat err=%v", err)
	}
}
