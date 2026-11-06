//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidence_RecallToCitedOriginalText(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	env := NewTestEnv(t)
	env.Init()
	gitCommit(t, env.RepoDir, "initial")
	path := filepath.Join(sessionDirFor(t, env.RepoDir), "evidence.jsonl")
	if err := os.WriteFile(path, []byte(transcript("unrelated opening", "quartzdecision: the original proposal", "quartzdecision: rejected; use the correction")), 0644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("checkpoint: %v %s", err, stderr)
	}
	before, _, err := env.RunCLI("query", "--sql", "SELECT count(*),sum(length(content)) FROM turns")
	if err != nil {
		t.Fatal(err)
	}
	recall, stderr, err := env.RunCLI("--json", "--limit", "1", "quartzdecision")
	if err != nil {
		t.Fatalf("recall: %v %s", err, stderr)
	}
	input := filepath.Join(env.RepoDir, "recall.json")
	if err := os.WriteFile(input, []byte(recall), 0644); err != nil {
		t.Fatal(err)
	}
	result, stderr, err := env.RunCLI("evidence", "--input", input, "--radius", "0", "--json")
	if err != nil {
		t.Fatalf("evidence: %v %s", err, stderr)
	}
	var out struct {
		Windows []struct {
			SessionID string `json:"session_id"`
			Turns     []struct {
				Index    int    `json:"index"`
				Content  string `json:"content"`
				Citation string `json:"citation"`
				Ts       string `json:"ts"`
			} `json:"turns"`
		} `json:"windows"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Windows) != 1 || len(out.Windows[0].Turns) != 2 {
		t.Fatalf("proposal and correction not both fetched: %s", result)
	}
	for _, turn := range out.Windows[0].Turns {
		if turn.Index == 0 || !strings.HasPrefix(turn.Content, "quartzdecision:") || !strings.HasPrefix(turn.Citation, out.Windows[0].SessionID+":t") || turn.Ts == "" {
			t.Fatalf("invalid source evidence: %+v", turn)
		}
	}
	after, _, err := env.RunCLI("query", "--sql", "SELECT count(*),sum(length(content)) FROM turns")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("evidence fetch changed the ledger: before=%q after=%q", before, after)
	}
}
