package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

// TestFormatDigest_MatchesRoutePy proves the in-binary digest is byte-identical
// to the skill's route.py: it feeds the SAME captured recall JSON to both and
// compares. Isolating the formatter from recall's (nondeterministic, warming)
// scoring is the whole point of driving from fixtures.
func TestFormatDigest_MatchesRoutePy(t *testing.T) {
	route := filepath.Join("skill", "skills", "rekal", "scripts", "route.py")
	if _, err := os.Stat(route); err != nil {
		t.Skipf("route.py not found: %v", err)
	}
	files, err := filepath.Glob(filepath.Join("testdata", "digest", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no digest testdata: %v", err)
	}

	// Shipped defaults: drop any ambient REKAL_HUNT_* so both sides agree.
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "REKAL_HUNT_") {
			env = append(env, e)
		}
	}

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var out search.Output
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: unmarshal: %v", f, err)
		}
		got, _ := formatDigest(&out)

		cmd := exec.Command("python3", route)
		cmd.Stdin = strings.NewReader(string(raw))
		cmd.Env = env
		wantBytes, _ := cmd.Output() // route.py exits 1 on SILENCE; stdout still captured
		want := string(wantBytes)

		if got != want {
			t.Errorf("%s digest mismatch:\n--- go ---\n%q\n--- route.py ---\n%q", filepath.Base(f), got, want)
		}
	}
}
