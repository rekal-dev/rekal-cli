package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestViewSession_MatchesGolden proves the in-binary session view reproduces
// the skill's view.py session output. Goldens generated from view.py on the
// fixture drill JSONs (testdata/view/{drill,drill_full}.json).
func TestViewSession_MatchesGolden(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"drill", "drill_full"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "view", name+".json"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		want, err := os.ReadFile(filepath.Join("testdata", "view", name+".golden"))
		if err != nil {
			t.Fatalf("read golden %s: %v", name, err)
		}
		var s sessionOutput
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		// view.py prints with a trailing newline; viewSession returns without.
		got := viewSession(&s) + "\n"
		if got != string(want) {
			t.Errorf("%s viewSession mismatch:\n--- got ---\n%s\n--- golden ---\n%s", name, got, string(want))
		}
	}
}

func TestViewRows(t *testing.T) {
	t.Parallel()
	cols := []string{"id", "n"}
	rows := []map[string]interface{}{
		{"id": "s1", "n": int64(3)},
		{"id": "s2", "n": nil},
		{"id": "s3", "n": "a\nb"},
	}
	got := viewRows(cols, rows)
	want := "id\tn\ns1\t3\ns2\t\ns3\ta\\nb"
	if got != want {
		t.Errorf("viewRows:\n got %q\nwant %q", got, want)
	}
	if viewRows(cols, nil) != "(no rows)" {
		t.Error("empty rows should be (no rows)")
	}
}

// TestViewSession_CommitsBlock covers the reverse link in the drill.
//
// Recall has gone commit → session for a long time (--commit). The drill was
// one-way: it handed the agent the reasoning with no pointer to the diff that
// reasoning became, so closing the loop meant a SQL query the agent had to
// know to write. The block is additive and appears only when the session
// actually produced commits, which is why adding it left the golden session
// view byte-identical.
func TestViewSession_CommitsBlock(t *testing.T) {
	t.Parallel()

	base := sessionOutput{
		Sid:   "s1",
		Turns: []turnOutput{{Index: 0, Role: "human", Content: "why did we drop batching"}},
	}

	// No commits: nothing added, and no stray heading.
	if got := viewSession(&base); strings.Contains(got, "commits:") {
		t.Errorf("a session with no commits must not grow a commits block:\n%s", got)
	}

	withCommits := base
	withCommits.Commits = []commitRef{
		{SHA: "abcdef1234567890", Subject: "fix: drop batching"},
		{SHA: "1234567890abcdef", Subject: "test: pin the retry path"},
	}
	got := viewSession(&withCommits)
	if !strings.Contains(got, "commits:") {
		t.Fatalf("commits block missing:\n%s", got)
	}
	if !strings.Contains(got, "abcdef12 fix: drop batching") {
		t.Errorf("want an abbreviated sha and its subject — the sha to run git show with, the subject to tell them apart without running it:\n%s", got)
	}
	if strings.Contains(got, "abcdef1234567890") {
		t.Errorf("the full sha is noise in a drill header; abbreviate it:\n%s", got)
	}

	// Many commits are capped rather than flooding the drill.
	many := base
	for i := 0; i < 14; i++ {
		many.Commits = append(many.Commits, commitRef{SHA: fmt.Sprintf("%08dxxxx", i), Subject: "c"})
	}
	gotMany := viewSession(&many)
	if !strings.Contains(gotMany, "(+4 more commits)") {
		t.Errorf("a 14-commit session must be capped with a remainder count, not listed in full:\n%s", gotMany)
	}
}
