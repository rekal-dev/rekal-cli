package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

// TestFormatDigest_MatchesGolden proves the in-binary digest reproduces the
// skill's route.py output. The .golden files were generated from route.py on
// the fixture recall JSONs (see testdata/digest); this test guards that
// formatDigest stays identical to that reference after route.py is removed.
func TestFormatDigest_MatchesGolden(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob(filepath.Join("testdata", "digest", "*.json"))
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no digest testdata: %v", err)
	}
	for _, in := range inputs {
		golden := strings.TrimSuffix(in, ".json") + ".golden"
		wantBytes, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("missing golden %s: %v", golden, err)
		}
		raw, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		var out search.Output
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: unmarshal: %v", in, err)
		}
		got, _ := formatDigest(&out)
		if got != string(wantBytes) {
			t.Errorf("%s digest mismatch:\n--- got ---\n%q\n--- golden ---\n%q", filepath.Base(in), got, string(wantBytes))
		}
	}
}
