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

// TestLineage_EmitsQueryAndCandidates verifies NDJSON events cover the
// query summary (weights + timings + counts) and per-session candidates
// with layer raw/norm/contrib lineage.
func TestLineage_EmitsQueryAndCandidates(t *testing.T) {
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

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("want candidate+query events, got %d lines: %s", len(lines), buf.String())
	}

	var sawQuery, sawCandidate bool
	for _, line := range lines {
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &head); err != nil {
			t.Fatalf("ndjson: %v (%s)", err, line)
		}
		switch head.Type {
		case "query":
			sawQuery = true
			var q LineageQuery
			if err := json.Unmarshal([]byte(line), &q); err != nil {
				t.Fatalf("query event: %v", err)
			}
			if q.Mode != "hybrid" || q.Query != "auth" {
				t.Fatalf("query event mismatch: %+v", q)
			}
			if q.TimingsMS["total"] < 0 || q.TimingsMS["bm25"] < 0 {
				t.Fatalf("timings missing: %+v", q.TimingsMS)
			}
			if q.Counts["candidates"] < 1 {
				t.Fatalf("counts: %+v", q.Counts)
			}
			if q.Weights.FacetBoost != 0.3 {
				t.Fatalf("weights snapshot lost facet_boost: %+v", q.Weights)
			}
		case "candidate":
			sawCandidate = true
			var c LineageCandidate
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				t.Fatalf("candidate event: %v", err)
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
		default:
			t.Fatalf("unexpected event type %q", head.Type)
		}
	}
	if !sawQuery || !sawCandidate {
		t.Fatalf("sawQuery=%v sawCandidate=%v; log=%s", sawQuery, sawCandidate, buf.String())
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
	on, err := Run(indexDB, Filters{Query: "token", Lineage: NewNDJSONLineage(&buf, 5)}, gitRoot, w, stubEmbedder{})
	if err != nil {
		t.Fatalf("on: %v", err)
	}
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
