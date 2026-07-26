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

// TestWithEvidence_DropsZeroEvidenceSeeds pins the digest's seed filter. A
// result can reach the window on max-normalized score alone — normalization is
// relative, so when every candidate is weak something still floats to the top
// with no absolute evidence behind it. On a real store those are harness echoes
// ("Reply with exactly: OK") spending slots in a 20-seed budget.
//
// The gate is exact zero on both absolute measures, never a confidence cutoff:
// a seed with any evidence at all still renders and the agent judges it from
// `conf=`. Anything else would be a corpus-tuned constant, which the soul
// forbids.
func TestWithEvidence_DropsZeroEvidenceSeeds(t *testing.T) {
	t.Parallel()

	in := []search.Result{
		{Sid: "s1", Score: 0.9, Confidence: 0.4, Mass: 2.1},
		{Sid: "noise", Score: 0.7},                // no confidence, no mass
		{Sid: "s2", Score: 0.5, Confidence: 0.01}, // tiny but real
		{Sid: "s3", Score: 0.4, Mass: 0.02},       // mass only
		{Sid: "alsonoise", Score: 0.3},            // no evidence
	}

	got := withEvidence(in)

	var ids []string
	for _, r := range got {
		ids = append(ids, r.Sid)
	}
	want := []string{"s1", "s2", "s3"}
	if len(ids) != len(want) {
		t.Fatalf("kept %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("kept[%d] = %q, want %q", i, ids[i], want[i])
		}
	}
}
