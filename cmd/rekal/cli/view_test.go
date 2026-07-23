package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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
