package search

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// TestLineage_OffMeansNoEvents confirms a nil Lineage emits nothing and
// ranking still returns results — the default path stays silent.
func TestLineage_OffMeansNoEvents(t *testing.T) {
	t.Parallel()
	indexDB := seedLineageCorpus(t)

	out, err := Run(indexDB, Filters{Query: "auth"}, t.TempDir(), DefaultWeights(), stubEmbedder{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Total == 0 {
		t.Fatal("expected at least one result")
	}
}

// TestLineage_EnvelopeAndEvents verifies NDJSON envelope (ts/v/run_id/event)
// plus query → candidate → result joined by run_id.
func TestLineage_EnvelopeAndEvents(t *testing.T) {
	t.Parallel()
	indexDB := seedLineageCorpus(t)

	var buf bytes.Buffer
	lin := NewNDJSONLineage(&buf, 10)
	out, err := Run(indexDB, Filters{Query: "auth", Lineage: lin}, t.TempDir(), DefaultWeights(), stubEmbedder{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Total == 0 {
		t.Fatal("expected results")
	}
	lin.FlushResult(1234)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("want query+candidate+result, got %d lines: %s", len(lines), buf.String())
	}

	var sawQuery, sawCandidate, sawResult bool
	var runID string
	for _, line := range lines {
		var head struct {
			TS    string `json:"ts"`
			V     int    `json:"v"`
			RunID string `json:"run_id"`
			Event string `json:"event"`
		}
		if err := json.Unmarshal([]byte(line), &head); err != nil {
			t.Fatalf("ndjson: %v (%s)", err, line)
		}
		if head.TS == "" || head.V != LineageSchemaVersion || head.RunID == "" {
			t.Fatalf("envelope incomplete: %+v (%s)", head, line)
		}
		if runID == "" {
			runID = head.RunID
		} else if head.RunID != runID {
			t.Fatalf("run_id mismatch: %q vs %q", runID, head.RunID)
		}
		switch head.Event {
		case "query":
			sawQuery = true
			var q struct {
				Query   string         `json:"query"`
				Mode    string         `json:"mode"`
				Weights LineageWeights `json:"weights"`
			}
			if err := json.Unmarshal([]byte(line), &q); err != nil {
				t.Fatalf("query: %v", err)
			}
			if q.Mode != "hybrid" || q.Query != "auth" {
				t.Fatalf("query mismatch: %+v", q)
			}
			if q.Weights.FacetBoost != 0.3 {
				t.Fatalf("weights snapshot lost facet_boost: %+v", q.Weights)
			}
		case "candidate":
			sawCandidate = true
			var c struct {
				SessionID    string             `json:"session_id"`
				RankPreGroup int                `json:"rank_pre_group"`
				Contrib      map[string]float64 `json:"contrib"`
				BestTurn     *LineageBestTurn   `json:"best_turn"`
			}
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				t.Fatalf("candidate: %v", err)
			}
			if c.SessionID == "" || c.RankPreGroup < 1 {
				t.Fatalf("candidate incomplete: %+v", c)
			}
			if _, ok := c.Contrib["bm25"]; !ok {
				t.Fatalf("contrib missing bm25: %+v", c.Contrib)
			}
			if c.BestTurn == nil || c.BestTurn.Role == "" {
				t.Fatalf("best_turn missing: %+v", c.BestTurn)
			}
		case "result":
			sawResult = true
			var r struct {
				Returned  []LineageReturned `json:"returned"`
				Counts    map[string]int    `json:"counts"`
				TimingsMS map[string]int64  `json:"timings_ms"`
				Tokens    *LineageTokens    `json:"tokens"`
			}
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				t.Fatalf("result: %v", err)
			}
			if len(r.Returned) < 1 || r.Returned[0].Rank != 1 {
				t.Fatalf("returned: %+v", r.Returned)
			}
			if r.Counts["after_group"] < 1 || r.Counts["candidates"] < 1 {
				t.Fatalf("counts: %+v", r.Counts)
			}
			if r.TimingsMS["total"] < 0 || r.TimingsMS["bm25"] < 0 {
				t.Fatalf("timings: %+v", r.TimingsMS)
			}
			if r.Tokens == nil || r.Tokens.PayloadBytes != 1234 {
				t.Fatalf("tokens.payload_bytes = %+v, want 1234", r.Tokens)
			}
		default:
			t.Fatalf("unexpected event %q", head.Event)
		}
	}
	if !sawQuery || !sawCandidate || !sawResult {
		t.Fatalf("sawQuery=%v sawCandidate=%v sawResult=%v; log=%s", sawQuery, sawCandidate, sawResult, buf.String())
	}
	if lin.RunID() != runID {
		t.Fatalf("RunID()=%q logged=%q", lin.RunID(), runID)
	}
}

// TestLineage_RankingIdenticalOnVsOff is the observe-only contract: enabling
// lineage must not change scores or result order.
func TestLineage_RankingIdenticalOnVsOff(t *testing.T) {
	t.Parallel()
	indexDB := seedLineageCorpus(t)
	w := DefaultWeights()
	gitRoot := t.TempDir()

	off, err := Run(indexDB, Filters{Query: "token"}, gitRoot, w, stubEmbedder{})
	if err != nil {
		t.Fatalf("off: %v", err)
	}
	var buf bytes.Buffer
	lin := NewNDJSONLineage(&buf, 5)
	on, err := Run(indexDB, Filters{Query: "token", Lineage: lin}, gitRoot, w, stubEmbedder{})
	if err != nil {
		t.Fatalf("on: %v", err)
	}
	lin.FlushResult(0)
	if off.Total != on.Total {
		t.Fatalf("total off=%d on=%d", off.Total, on.Total)
	}
	for i := range off.Results {
		if off.Results[i].SessionID != on.Results[i].SessionID || off.Results[i].Score != on.Results[i].Score {
			t.Fatalf("result[%d] diverged: off=%+v on=%+v", i, off.Results[i], on.Results[i])
		}
	}
	if buf.Len() == 0 {
		t.Fatal("expected lineage events when enabled")
	}
}

func seedLineageCorpus(t *testing.T) *sql.DB {
	t.Helper()
	indexDB := openTempIndexDB(t)
	if err := db.LoadFTSExtension(indexDB); err != nil {
		t.Skipf("FTS extension unavailable: %v", err)
	}
	for _, sid := range []string{"s-auth", "s-db"} {
		if _, err := indexDB.Exec(
			`INSERT INTO session_facets (session_id, actor_type, captured_at, turn_count, tool_call_count, file_count)
			 VALUES ($1, 'human', '2026-01-01T00:00:00Z', 3, 0, 0)`,
			sid,
		); err != nil {
			t.Fatalf("seed facet: %v", err)
		}
	}
	for _, r := range []struct {
		id, sid, role, content string
		idx                    int
	}{
		{"t1", "s-auth", "human", "how does JWT auth token expiry work", 0},
		{"t2", "s-auth", "assistant", "check the auth middleware for token refresh", 1},
		{"t3", "s-auth", "human_steering", "no, use the existing auth helper instead", 2},
		{"t4", "s-db", "human", "migrate the database schema", 0},
		{"t5", "s-db", "assistant", "run the migration tool", 1},
	} {
		if _, err := indexDB.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, $3, $4, $5, '')`,
			r.id, r.sid, r.idx, r.role, r.content,
		); err != nil {
			t.Fatalf("seed turn: %v", err)
		}
	}
	if err := db.CreateFTSIndex(indexDB); err != nil {
		t.Fatalf("CreateFTSIndex: %v", err)
	}
	return indexDB
}
