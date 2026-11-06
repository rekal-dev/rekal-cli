package evidence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

func evidenceTestIndex(t *testing.T) *sql.DB {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0755); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := db.InitIndexSchema(d); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO session_facets(session_id,actor_type,captured_at) VALUES ('session-a','human','2026-11-06T10:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	return d
}

func evidenceTestTurn(t *testing.T, d *sql.DB, index int, text string) {
	t.Helper()
	if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ($1,'session-a',$2,'assistant',$3,'2026-11-06T10:00:00Z')`, fmt.Sprint(index), index, text); err != nil {
		t.Fatal(err)
	}
}

func TestEvidence_MergesOverlapsAndPreservesCitations(t *testing.T) {
	t.Parallel()
	d := evidenceTestIndex(t)
	for i := 0; i < 9; i++ {
		evidenceTestTurn(t, d, i, fmt.Sprintf("original %d", i))
	}
	req := Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 2, EndTurn: 2}, {SessionID: "session-a", StartTurn: 3, EndTurn: 3}}}
	out, err := Fetch(d, req, 1, 16000, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Windows) != 1 || len(out.Windows[0].Turns) != 4 {
		t.Fatalf("overlaps not merged: %+v", out)
	}
	for i, turn := range out.Windows[0].Turns {
		want := i + 1
		if turn.Index != want || turn.Content != fmt.Sprintf("original %d", want) || turn.Citation != fmt.Sprintf("session-a:t%d", want) || turn.Ts != "2026-11-06T10:00:00Z" {
			t.Fatalf("source changed: %+v", turn)
		}
	}
	gap, err := Fetch(d, Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 8, EndTurn: 8}, {SessionID: "session-a", StartTurn: 0, EndTurn: 0}}}, 0, 16000, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(gap.Windows[0].Gaps) != 1 || gap.Windows[0].Gaps[0].StartTurn != 1 || gap.Windows[0].Gaps[0].EndTurn != 7 {
		t.Fatalf("missing visible gap: %+v", gap)
	}
}

func TestEvidence_LongUTF8TurnPagesWithoutLoss(t *testing.T) {
	t.Parallel()
	d := evidenceTestIndex(t)
	original := strings.Repeat("证据", 100) + " final answer"
	evidenceTestTurn(t, d, 0, original)
	req := Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 0, EndTurn: 0}}}
	var reconstructed strings.Builder
	for page := 0; page < 20; page++ {
		out, err := Fetch(d, req, 0, 128, 3)
		if err != nil {
			t.Fatal(err)
		}
		if out.ContentBytes > 128 {
			t.Fatal("byte budget exceeded")
		}
		for _, w := range out.Windows {
			for _, turn := range w.Turns {
				if !utf8.ValidString(turn.Content) {
					t.Fatal("split UTF-8 rune")
				}
				reconstructed.WriteString(turn.Content)
			}
		}
		if out.Next == nil {
			break
		}
		encoded, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		var page struct {
			Next *Request `json:"next"`
		}
		err = json.Unmarshal(encoded, &page)
		req = *page.Next
		if err != nil {
			t.Fatal(err)
		}
		if req.Radius == nil || *req.Radius != 0 {
			t.Fatal("continuation lost radius")
		}
	}
	if reconstructed.String() != original {
		t.Fatalf("pagination changed source: got %d bytes, want %d", reconstructed.Len(), len(original))
	}
}

func TestEvidence_MatchedTextPrecedesLargeNeighborBudget(t *testing.T) {
	t.Parallel()
	d := evidenceTestIndex(t)
	evidenceTestTurn(t, d, 0, strings.Repeat("x", 20000))
	evidenceTestTurn(t, d, 1, "supporting evidence")
	out, err := Fetch(d, Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 1, EndTurn: 1}}}, 1, 128, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range out.Windows {
		for _, turn := range w.Turns {
			if turn.Index == 1 && turn.Content == "supporting evidence" {
				found = true
			}
		}
	}
	if !found || !out.Truncated || out.Next == nil {
		t.Fatalf("context displaced the match or truncation was hidden: %+v", out)
	}
}

func TestEvidence_MissingAnchorDoesNotInventNeighborEvidence(t *testing.T) {
	t.Parallel()
	d := evidenceTestIndex(t)
	evidenceTestTurn(t, d, 0, "neighbor only")
	out, err := Fetch(d, Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 1, EndTurn: 1}}}, 2, 128, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Windows) != 0 || len(out.Warnings) != 1 {
		t.Fatalf("missing anchor returned unrelated context: %+v", out)
	}
}

func TestEvidence_ResumeRejectsChangedSourceWindow(t *testing.T) {
	t.Parallel()
	d := evidenceTestIndex(t)
	evidenceTestTurn(t, d, 0, strings.Repeat("evidence ", 100))
	out, err := Fetch(d, Request{Anchors: []search.Anchor{{SessionID: "session-a", StartTurn: 0, EndTurn: 0}}}, 2, 128, 10)
	if err != nil || out.Next == nil {
		t.Fatalf("first page: %v %+v", err, out)
	}
	evidenceTestTurn(t, d, 1, "newly appended context")
	if _, err := Fetch(d, *out.Next, 2, 128, 10); err == nil || !strings.Contains(err.Error(), "source windows changed") {
		t.Fatalf("stale cursor accepted: %v", err)
	}
}
