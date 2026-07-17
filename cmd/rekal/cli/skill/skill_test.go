package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAll_UnifiedSkill verifies the collapsed suite: one skill named rekal,
// well-formed tip, and every dispatch target (references + scripts) present.
func TestAll_UnifiedSkill(t *testing.T) {
	t.Parallel()

	skills := All()
	if len(skills) != 1 {
		t.Fatalf("expected exactly 1 skill after suite collapse, got %d", len(skills))
	}
	s := skills[0]
	if s.Name != "rekal" {
		t.Fatalf("skill name = %q, want rekal", s.Name)
	}
	if !strings.HasPrefix(s.Content, "---\n") {
		t.Fatal("missing front-matter")
	}
	if !strings.Contains(s.Content, "name: rekal\n") {
		t.Fatal("front-matter name mismatch")
	}
	if !strings.Contains(s.Content, "description:") {
		t.Fatal("missing description")
	}

	wantFiles := []string{
		"SKILL.md",
		"scripts/hunt-gate.py",
		"scripts/recall-route.py",
		"scripts/why-trail-gate.py",
		"scripts/map-fresh.sh",
		"scripts/map-write-watermark.sh",
		"scripts/wiki-branch-gate.sh",
		"references/hunt.md",
		"references/why.md",
		"references/mine.md",
		"references/map.md",
		"references/provenance.md",
		"references/analytics.md",
		"references/wiki.md",
		"references/reference.md",
	}
	for _, rel := range wantFiles {
		if _, ok := s.Files[rel]; !ok {
			t.Errorf("missing embedded file %s", rel)
		}
	}
	// Tip must name the modules so progressive disclosure stays wired.
	for _, needle := range []string{
		"scripts/recall-route.py",
		"scripts/why-trail-gate.py",
		"scripts/map-fresh.sh",
		"scripts/wiki-branch-gate.sh",
		"references/hunt.md",
		"references/analytics.md",
	} {
		if !strings.Contains(s.Content, needle) {
			t.Errorf("SKILL.md tip must mention %s", needle)
		}
	}
	for _, name := range LegacyNames {
		if name == "" {
			t.Fatal("empty legacy name")
		}
	}
}

func TestIsScript(t *testing.T) {
	t.Parallel()
	if !IsScript("scripts/hunt-gate.py") {
		t.Fatal("scripts/ should be executable")
	}
	if IsScript("references/hunt.md") {
		t.Fatal("references/ should not be marked script")
	}
}

func writeScript(t *testing.T, rel string) string {
	t.Helper()
	skills := All()
	if _, ok := skills[0].Files[rel]; !ok {
		t.Fatalf("missing %s", rel)
	}
	dir := t.TempDir()
	// Preserve sibling scripts when recall-route needs hunt-gate next door.
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range skills[0].Files {
		if !strings.HasPrefix(name, "scripts/") {
			continue
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, rel)
}

func TestHuntGateScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/hunt-gate.py")

	passJSON := `{"results":[{"confidence":0.82,"mass":5.6,"score":0.99},{"confidence":0.4,"mass":2.0,"score":0.5}],"knowledge":[]}`
	cmd := exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(passJSON)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PASS_EPISODE case failed: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "PASS_EPISODE") {
		t.Fatalf("want PASS_EPISODE, got %s", out)
	}

	// Junk: high max-normalized score, low absolute confidence + weak mass.
	junkJSON := `{"results":[{"confidence":0.48,"mass":2.59,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(junkJSON)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("junk should SILENCE, got %s", out)
	}
	if !strings.Contains(string(out), "SILENCE") {
		t.Fatalf("want SILENCE, got %s", out)
	}

	// BUG 11: offtopic confidence (~0.63) must not soft-pass even with strong mass + gap.
	offtopicJSON := `{"results":[{"confidence":0.63,"mass":4.0,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(offtopicJSON)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("offtopic conf 0.63 should SILENCE, got %s", out)
	}
	if !strings.Contains(string(out), "SILENCE") {
		t.Fatalf("want SILENCE for offtopic, got %s", out)
	}

	// With confidence present, never fall back to max-norm score for gating.
	scoreOnlyMixed := `{"results":[{"confidence":0.5,"score":1.19},{"score":0.95}],"knowledge":[]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(scoreOnlyMixed)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("confidence set must not gate on score, got %s", out)
	}

	silenceJSON := `{"results":[{"confidence":0.5,"score":0.5},{"confidence":0.49,"score":0.49}],"knowledge":[]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(silenceJSON)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("SILENCE case should exit non-zero, got %s", out)
	}

	// Confident episode wins even when knowledge docs are present.
	strongPlusKnow := `{"results":[{"confidence":0.8,"mass":5.0,"score":1.08},{"confidence":0.3,"score":0.5}],"knowledge":[{"path":"docs/x.md"}]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(strongPlusKnow)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("strong episode + knowledge should PASS_EPISODE: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "PASS_EPISODE") {
		t.Fatalf("want PASS_EPISODE, got %s", out)
	}

	// Knowledge is only a fallback when the episode gate fails.
	weakPlusKnow := `{"results":[{"confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[{"path":"docs/x.md"}]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(weakPlusKnow)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("weak episode + knowledge should exit 3, got ok: %s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("want exit 3, got %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "ROUTE_KNOWLEDGE") {
		t.Fatalf("want ROUTE_KNOWLEDGE, got %s", out)
	}
}

func TestRecallRouteScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/recall-route.py")

	knowJSON := `{"results":[{"score":0.4}],"knowledge":[{"path":"docs/x.md"}]}`
	cmd := exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(knowJSON)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("KNOWLEDGE should exit 0: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "KNOWLEDGE") {
		t.Fatalf("want KNOWLEDGE, got %s", out)
	}
	if strings.Contains(string(out), "INJECT") {
		t.Fatalf("weak episode + knowledge must not INJECT: %s", out)
	}

	injectJSON := `{"results":[{"confidence":0.8,"mass":5.5,"score":0.95},{"confidence":0.4,"score":0.5}],"knowledge":[]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(injectJSON)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("INJECT should exit 0: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "INJECT") {
		t.Fatalf("want INJECT, got %s", out)
	}

	// PKYC-class bug: confident session must INJECT despite prose hits.
	bothJSON := `{"results":[{"confidence":0.85,"mass":6.0,"score":1.08},{"confidence":0.4,"score":0.5}],"knowledge":[{"path":"docs/a.md"},{"path":"docs/b.md"}]}`
	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(bothJSON)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("strong episode + knowledge should INJECT: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "INJECT") {
		t.Fatalf("want INJECT, got %s", out)
	}
}

func TestWhyTrailGateScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/why-trail-gate.py")

	rows := strings.Repeat("session\t1\thuman\tbecause\n", 12)
	cmd := exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(rows)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PASS expected: %v (%s)", err, out)
	}

	cmd = exec.Command("python3", path)
	cmd.Stdin = strings.NewReader("one\ntwo\n")
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("under_gathered should fail, got %s", out)
	}
	if !strings.Contains(string(out), "under_gathered") {
		t.Fatalf("want under_gathered, got %s", out)
	}
}
