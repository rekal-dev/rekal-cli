package session

import (
	"os"
	"testing"
)

// realFixturesDir holds sanitized copies of real Claude Code session
// transcripts captured on a developer machine — trunk sessions, subagent
// transcripts, and their meta.json sidecars — preserving the exact directory
// layout and JSONL field shapes Claude Code actually writes. See
// testdata/real/README.md for what they are and how to refresh them.
//
// The synthetic-fixture-vs-real-data gap has shipped two bugs despite green
// suites (migration and checkpoint bugs, both only caught by manually
// validating against real transcripts — see docs/REVIEW_2026-07-03.md).
// These tests are that manual validation step, automated and checked in,
// instead of a one-off pass repeated from scratch the next time something
// slips through.
const realFixturesDir = "testdata/real"

// TestRealFixtures_DiscoverAndParseWithoutError runs the full discovery +
// parse pipeline (the same path checkpoint.go uses in production) against
// real transcript shapes. A hand-written synthetic fixture only exercises
// the fields its author remembered to include; real transcripts carry
// harness noise (isMeta wrappers, queue-operation entries, sidechain
// duplicates, deeply nested tool_use/tool_result chains) that a synthetic
// fixture easily omits by accident.
func TestRealFixtures_DiscoverAndParseWithoutError(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat(realFixturesDir); os.IsNotExist(err) {
		t.Skip("no real fixtures checked in yet — see testdata/real/README.md")
	}

	refs, err := discoverSessionRefs(realFixturesDir)
	if err != nil {
		t.Fatalf("discoverSessionRefs: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected at least one discovered session ref from real fixtures")
	}

	adapter := &ClaudeAdapter{}
	var trunkCount, subagentCount int

	for _, ref := range refs {
		payload, err := adapter.Parse(ref)
		if err != nil {
			t.Fatalf("Parse(%s): %v", ref.Path, err)
		}
		if payload == nil {
			// Empty file — valid, e.g. a session that captured no turns.
			continue
		}
		if payload.SessionID == "" {
			t.Errorf("Parse(%s): empty SessionID", ref.Path)
		}
		if payload.Source != "claude" {
			t.Errorf("Parse(%s): Source = %q, want claude", ref.Path, payload.Source)
		}

		if ref.ParentPath == "" {
			trunkCount++
			if payload.ActorType != "human" {
				t.Errorf("Parse(%s): trunk ActorType = %q, want human", ref.Path, payload.ActorType)
			}
			continue
		}

		subagentCount++
		if payload.ActorType != "agent" {
			t.Errorf("Parse(%s): subagent ActorType = %q, want agent", ref.Path, payload.ActorType)
		}
		if payload.AgentID == "" {
			t.Errorf("Parse(%s): subagent transcript has empty AgentID", ref.Path)
		}
		if payload.ParentSessionPath != ref.ParentPath {
			t.Errorf("Parse(%s): ParentSessionPath = %q, want %q", ref.Path, payload.ParentSessionPath, ref.ParentPath)
		}
	}

	if trunkCount == 0 {
		t.Error("expected at least one trunk session among real fixtures")
	}
	if subagentCount == 0 {
		t.Error("expected at least one subagent transcript among real fixtures — that's the shape most likely to regress silently")
	}
}

// TestRealFixtures_SubagentMetaSidecarFieldsCaptured pins the real meta.json
// sidecar shape (agentType/description/spawnDepth — see claude.go's
// subagentMeta doc) against one specific captured fixture, so a future
// change to the sidecar field names silently reverting to the old, wrong
// guess (agentId/teamName) would be caught here instead of only in a
// hand-written unit test.
func TestRealFixtures_SubagentMetaSidecarFieldsCaptured(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat(realFixturesDir); os.IsNotExist(err) {
		t.Skip("no real fixtures checked in yet — see testdata/real/README.md")
	}

	const agentFile = realFixturesDir + "/e4ff45ed-2313-4624-9a11-daf9582c4a20/subagents/agent-a2968f2134831a849.jsonl"
	const parentPath = realFixturesDir + "/e4ff45ed-2313-4624-9a11-daf9582c4a20.jsonl"

	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		t.Skip("pinned fixture not present — real fixtures may have been refreshed without it")
	}

	adapter := &ClaudeAdapter{}
	payload, err := adapter.Parse(SessionRef{Path: agentFile, ParentPath: parentPath})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if payload.AgentType != "general-purpose" {
		t.Errorf("AgentType = %q, want general-purpose", payload.AgentType)
	}
	if payload.Description == "" {
		t.Error("Description is empty, want the real captured description")
	}
	if payload.SpawnDepth != 1 {
		t.Errorf("SpawnDepth = %d, want 1", payload.SpawnDepth)
	}
	if payload.TeamName != "" {
		t.Errorf("TeamName = %q, want empty — this sidecar carries no teamName field", payload.TeamName)
	}
}
