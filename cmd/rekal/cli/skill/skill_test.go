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
		"scripts/route.py",
		"scripts/map.sh",
		"scripts/wiki-gate.sh",
		"references/ledger.md",
		"references/map.md",
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
		"scripts/route.py",
		"scripts/map.sh",
		"scripts/wiki-gate.sh",
		"references/ledger.md",
		"references/map.md",
		"references/reference.md",
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
	if !IsScript("scripts/route.py") {
		t.Fatal("scripts/ should be executable")
	}
	if IsScript("references/ledger.md") {
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

// run feeds JSON to route.py and returns combined output + exit code.
func runRoute(t *testing.T, path, stdin string) (string, int) {
	t.Helper()
	cmd := exec.Command("python3", path)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run route.py: %v (%s)", err, out)
		}
	}
	return string(out), code
}

func TestRouteScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/route.py")

	// Confident hit -> INJECT. Scores are the script's gating inputs, not emitted
	// to the agent's context (content-first), so the output carries no mass/conf.
	out, code := runRoute(t, path, `{"results":[{"confidence":0.82,"mass":5.6,"score":0.99},{"confidence":0.4,"mass":2.0,"score":0.5}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") || strings.Contains(out, "mass=") || strings.Contains(out, "conf=") {
		t.Fatalf("want INJECT exit 0 with no scores in output, got code=%d %s", code, out)
	}

	// Confident but lexically thin (dialogue) -> still INJECT, NOT silenced.
	// The chat fix: low mass is never a veto (it gated nothing, and isn't emitted).
	out, code = runRoute(t, path, `{"results":[{"confidence":0.82,"mass":1.4,"score":0.99},{"confidence":0.3,"mass":1.0,"score":0.4}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("want INJECT exit 0 (thin chat hit not silenced), got code=%d %s", code, out)
	}

	// Junk: high max-norm score, low absolute confidence -> SILENCE.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.48,"mass":2.59,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("junk should SILENCE, got code=%d %s", code, out)
	}

	// Offtopic confidence (~0.63) must not soft-pass even with strong mass + gap.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.63,"mass":4.0,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("offtopic conf 0.63 should SILENCE, got code=%d %s", code, out)
	}

	// With confidence present, never fall back to max-norm score for gating.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.5,"score":1.19},{"score":0.95}],"knowledge":[]}`)
	if code == 0 {
		t.Fatalf("confidence set must not gate on score, got %s", out)
	}

	// Offtopic on a modern store: omitempty drops all-zero confidence, so no
	// result carries a confidence key. Must SILENCE — not INJECT on score/gap.
	out, code = runRoute(t, path, `{"results":[{"session_id":"x","score":0.65},{"session_id":"y","score":0.43}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("missing-confidence offtopic set must SILENCE, got code=%d %s", code, out)
	}

	// Knowledge is the fallback when the episode gate fails -> KNOWLEDGE, exit 0.
	// Per-file path=score distribution is reported verbatim (no tuned floor).
	out, code = runRoute(t, path, `{"results":[{"confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[{"path":"docs/x.md","score":0.72}]}`)
	if code != 0 || !strings.Contains(out, "KNOWLEDGE") || !strings.Contains(out, "docs/x.md=0.72") {
		t.Fatalf("weak episode + knowledge should KNOWLEDGE docs/x.md=0.72 exit 0, got code=%d %s", code, out)
	}
	if strings.Contains(out, "INJECT") {
		t.Fatalf("weak episode + knowledge must not INJECT: %s", out)
	}

	// Confident episode wins even when knowledge docs are present.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.85,"mass":6.0,"score":1.08},{"confidence":0.4,"score":0.5}],"knowledge":[{"path":"docs/a.md"},{"path":"docs/b.md"}]}`)
	if code != 0 || !strings.Contains(out, "INJECT") {
		t.Fatalf("strong episode + knowledge should INJECT, got code=%d %s", code, out)
	}

	// A low knowledge score is REPORTED, not silenced on a tuned floor: the score
	// is a signal the agent judges. SOUL.md bans a corpus-tuned KNOWLEDGE floor.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[{"path":"docs/x.md","score":0.12}]}`)
	if code != 0 || !strings.Contains(out, "KNOWLEDGE") || !strings.Contains(out, "docs/x.md=0.12") {
		t.Fatalf("low knowledge score must be reported (KNOWLEDGE docs/x.md=0.12), not silenced, got code=%d %s", code, out)
	}

	// A retryable semantic-warming status adds a trailing note after the verdict
	// (line 1 stays the verdict), telling the agent to re-run for full quality.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.82,"mass":5,"score":1}],"knowledge":[],"semantic":{"status":"warming","retryable":true}}`)
	if code != 0 || !strings.Contains(out, "INJECT") || !strings.Contains(out, "SEMANTIC warming") {
		t.Fatalf("warming status should add a SEMANTIC warming note, got code=%d %s", code, out)
	}
	if !strings.HasPrefix(strings.SplitN(out, "\n", 2)[0], "INJECT ") {
		t.Fatalf("verdict must stay on line 1, got: %q", strings.SplitN(out, "\n", 2)[0])
	}
	// No semantic field (or not retryable) -> no note.
	out, _ = runRoute(t, path, `{"results":[{"confidence":0.82,"mass":5,"score":1}],"knowledge":[]}`)
	if strings.Contains(out, "SEMANTIC warming") {
		t.Fatalf("no semantic field must not emit a warming note: %s", out)
	}

	// Seed window: the digest shows up to DIGEST_WINDOW=20 candidates each with a
	// snippet (no scores), then summarizes the rest as "(+N more)", so an INJECT
	// stays cost-bounded regardless of -n. 30 results -> 20 shown + "(+10 more".
	big := `{"results":[{"session_id":"top","confidence":0.82,"mass":5,"score":1,"snippet":"hello world"}` +
		strings.Repeat(`,{"session_id":"x","confidence":0.5,"mass":1,"score":1,"snippet":"filler text"}`, 29) +
		`],"knowledge":[]}`
	out, code = runRoute(t, path, big)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") || !strings.Contains(out, "(+10 more") {
		t.Fatalf("digest should show a 20-candidate seed window and (+10 more), got: %s", out)
	}
	if strings.Contains(out, "conf=") || strings.Contains(out, "(0.5)") {
		t.Fatalf("seed digest must not print per-candidate scores: %s", out)
	}
	if n := strings.Count(out, "\"filler text\""); n > 20 { // capped at the window
		t.Fatalf("seed window not capped at 20, printed %d snippets: %s", n, out)
	}

	// Episode gate fails AND no knowledge at all -> machine SILENCE, exit 1.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("weak episode + no knowledge should SILENCE, got code=%d %s", code, out)
	}
}
