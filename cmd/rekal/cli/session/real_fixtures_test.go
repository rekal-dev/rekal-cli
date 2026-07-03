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
