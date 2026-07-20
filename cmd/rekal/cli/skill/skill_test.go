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
		"scripts/view.py",
		"scripts/find.py",
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
		"scripts/view.py",
		"scripts/find.py",
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
	// Drop harness overrides so shipped defaults are what we assert (ambient
	// REKAL_HUNT_* from industry-bench shells must not leak into unit tests).
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "REKAL_HUNT_") {
			continue
		}
		filtered = append(filtered, e)
	}
	cmd.Env = filtered
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

	// Confident hit -> INJECT. Confidence is emitted (header top=/gap= and per-row
	// conf=) so the agent can weigh seeds; mass stays inside the script.
	out, code := runRoute(t, path, `{"results":[{"session_id":"s1","confidence":0.82,"mass":5.6,"score":0.99},{"session_id":"s2","confidence":0.4,"mass":2.0,"score":0.5}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT top=0.82 gap=0.42 ") {
		t.Fatalf("want INJECT top/gap header exit 0, got code=%d %s", code, out)
	}
	if strings.Contains(out, "mass=") {
		t.Fatalf("mass must stay inside the script, got %s", out)
	}
	if !strings.Contains(out, "s1 conf=0.82") {
		t.Fatalf("want per-seed confidence, got %s", out)
	}

	// Prefer query-time sid over the full ULID in the digest.
	out, code = runRoute(t, path, `{"results":[{"session_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","sid":"s3","confidence":0.9,"mass":5,"score":1}],"knowledge":[]}`)
	if code != 0 || !strings.Contains(out, "s3 conf=0.90") || strings.Contains(out, "01ARZ3NDEKTSV4RRFFQ69G5FAV") {
		t.Fatalf("want digest to prefer sid, got code=%d %s", code, out)
	}

	// Confident but lexically thin (dialogue) -> still INJECT, NOT silenced.
	// The chat fix: low mass is never a veto (it gated nothing, and isn't emitted).
	out, code = runRoute(t, path, `{"results":[{"confidence":0.82,"mass":1.4,"score":0.99},{"confidence":0.3,"mass":1.0,"score":0.4}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("want INJECT exit 0 (thin chat hit not silenced), got code=%d %s", code, out)
	}

	// Grey-band confidence (old "junk" / dialogue range) injects with conf=
	// for the agent to weigh — the script suggests, it does not hard-veto.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.48,"mass":2.59,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT top=0.48 ") {
		t.Fatalf("grey-band conf should INJECT (suggest), got code=%d %s", code, out)
	}

	// Mid confidence with gap still injects under the low floor.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.63,"mass":4.0,"score":1.19},{"confidence":0.45,"mass":2.4,"score":0.9}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("conf 0.63 should INJECT under low floor, got code=%d %s", code, out)
	}

	// With confidence present, gate on confidence — never max-norm score alone.
	// 0.5 clears the low floor; missing-confidence offtopic (below) still silences.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.5,"score":1.19},{"score":0.95}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT top=0.50 ") {
		t.Fatalf("confidence-present hit should INJECT on conf, got code=%d %s", code, out)
	}

	// Offtopic on a modern store: omitempty drops all-zero confidence, so no
	// result carries a confidence key. Must SILENCE — not INJECT on score/gap.
	out, code = runRoute(t, path, `{"results":[{"session_id":"x","score":0.65},{"session_id":"y","score":0.43}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("missing-confidence offtopic set must SILENCE, got code=%d %s", code, out)
	}

	// Low episode floor + knowledge: inclusive INJECT with trailing KNOWLEDGE.
	out, code = runRoute(t, path, `{"results":[{"session_id":"ep","confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[{"path":"docs/x.md","score":0.72}]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("episode above low floor + knowledge should INJECT, got code=%d %s", code, out)
	}
	if !strings.Contains(out, "KNOWLEDGE") || !strings.Contains(out, "docs/x.md=0.72") {
		t.Fatalf("want trailing KNOWLEDGE docs/x.md=0.72, got %s", out)
	}

	// Inclusive: strong episode + knowledge emits INJECT (line 1) and a KNOWLEDGE
	// line — mixed questions can need both substrates.
	out, code = runRoute(t, path, `{"results":[{"session_id":"ep","confidence":0.85,"mass":6.0,"score":1.08},{"confidence":0.4,"score":0.5}],"knowledge":[{"path":"docs/a.md","score":0.91},{"path":"docs/b.md","score":0.55}]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("strong episode + knowledge should INJECT on line 1, got code=%d %s", code, out)
	}
	if !strings.Contains(out, "KNOWLEDGE") || !strings.Contains(out, "docs/a.md=0.91") {
		t.Fatalf("strong episode + knowledge must also report KNOWLEDGE, got %s", out)
	}

	// Below the super-low knowledge report floor: omit the KNOWLEDGE line
	// (junk marker noise) but still INJECT the episode.
	out, code = runRoute(t, path, `{"results":[{"session_id":"ep","confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[{"path":"docs/x.md","score":0.12}]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("episode should still INJECT, got code=%d %s", code, out)
	}
	if strings.Contains(out, "KNOWLEDGE") {
		t.Fatalf("knowledge below report floor must be omitted, got %s", out)
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

	// Seed window: digest shows up to DIGEST_WINDOW=20 candidates each with
	// conf= + snippet, then "(+N more)". 30 results -> 20 shown + "(+10 more".
	big := `{"results":[{"session_id":"top","confidence":0.82,"mass":5,"score":1,"snippet":"hello world"}` +
		strings.Repeat(`,{"session_id":"x","confidence":0.5,"mass":1,"score":1,"snippet":"filler text"}`, 29) +
		`],"knowledge":[]}`
	out, code = runRoute(t, path, big)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") || !strings.Contains(out, "(+10 more") {
		t.Fatalf("digest should show a 20-candidate seed window and (+10 more), got: %s", out)
	}
	if !strings.Contains(out, "top conf=0.82") || !strings.Contains(out, "x conf=0.50") {
		t.Fatalf("seed digest must print per-candidate confidence, got: %s", out)
	}
	if n := strings.Count(out, "\"filler text\""); n > 20 { // capped at the window
		t.Fatalf("seed window not capped at 20, printed %d snippets: %s", n, out)
	}

	// Episode above the low floor + no knowledge -> INJECT (agent weighs conf=).
	out, code = runRoute(t, path, `{"results":[{"session_id":"ep","confidence":0.4,"mass":2.0,"score":0.95}],"knowledge":[]}`)
	if code != 0 || !strings.HasPrefix(out, "INJECT ") {
		t.Fatalf("conf 0.40 should INJECT under low floor, got code=%d %s", code, out)
	}

	// Truly below the super-low floor + no knowledge -> machine SILENCE.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.10,"mass":2.0,"score":0.95}],"knowledge":[]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("conf 0.10 + no knowledge should SILENCE, got code=%d %s", code, out)
	}

	// Knowledge alone below report floor (no episode) -> SILENCE, not KNOWLEDGE.
	out, code = runRoute(t, path, `{"results":[{"confidence":0.10,"score":0.9}],"knowledge":[{"path":"CLAUDE.md","score":0.09}]}`)
	if code == 0 || !strings.Contains(out, "SILENCE") {
		t.Fatalf("junk knowledge alone should SILENCE, got code=%d %s", code, out)
	}
}

func TestViewScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/view.py")

	session := `{
	  "session_id":"s1",
	  "author":"a@b.c",
	  "actor":"human",
	  "branch":"main",
	  "captured_at":"2026-01-01T00:00:00Z",
	  "total_turns":3,
	  "turns":[
	    {"index":1,"role":"human","content":"hello world","ts":"2023-05-08T10:00:00Z"},
	    {"index":2,"role":"assistant","content":"hi there","ts":"2023-05-08T10:01:00Z"}
	  ]
	}`
	out, code := runRoute(t, path, session)
	if code != 0 {
		t.Fatalf("view session exit: %d %s", code, out)
	}
	if strings.Contains(out, `"session_id"`) || strings.Contains(out, "captured_at") {
		t.Fatalf("view must strip JSON chrome, got %s", out)
	}
	if !strings.Contains(out, "s1 t1-2") || !strings.Contains(out, "h=human a=assistant") {
		t.Fatalf("want compact header with role legend, got %s", out)
	}
	if !strings.Contains(out, "t1 2023-05-08T10:00:00Z h: hello world") {
		t.Fatalf("want first turn full timestamp, got %s", out)
	}
	// Same date as previous → shorten to …time (no repeated date).
	if !strings.Contains(out, "t2 …10:01:00Z a: hi there") {
		t.Fatalf("want same-date timestamp shortened with …, got %s", out)
	}

	rows := "{\"a\":1,\"b\":\"x\"}\n{\"a\":2,\"b\":\"y\"}\n"
	out, code = runRoute(t, path, rows)
	if code != 0 || !strings.Contains(out, "a\tb") || !strings.Contains(out, "1\tx") {
		t.Fatalf("want TSV rows, got code=%d %s", code, out)
	}

	// Engine error text on stdin (2>&1) is forwarded verbatim, exit 2 — never
	// swallowed as an empty set (wrongful absence).
	out, code = runRoute(t, path, `rekal: Binder Error: Referenced column "notacol" not found`)
	if code != 2 || !strings.Contains(out, "Binder Error") {
		t.Fatalf("want engine error forwarded exit 2, got code=%d %s", code, out)
	}
}

// TestFindScript drives find.py against a stub rekal (REKAL_BIN) so the
// enumeration sweep is tested without a real store.
func TestFindScript(t *testing.T) {
	t.Parallel()
	path := writeScript(t, "scripts/find.py")

	stub := filepath.Join(t.TempDir(), "rekal-stub")
	rowsOut := `{"session_id":"01ABC","turn_index":3,"ts":"2023-05-08 10:00:00","role":"human","content":"I adopted a turtle named Fred"}
{"session_id":"01DEF","turn_index":9,"ts":"2023-06-01 11:00:00","role":"assistant","content":"the turtle tank was cleaned"}`
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ncat <<'EOF'\n"+rowsOut+"\nEOF\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	runFind := func(bin string, args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command("python3", append([]string{path}, args...)...)
		cmd.Env = append(os.Environ(), "REKAL_BIN="+bin)
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("run find.py: %v (%s)", err, out)
			}
		}
		return string(out), code
	}

	// Complete sweep: every mention, one line each, plus the total.
	out, code := runFind(stub, "turtle")
	if code != 0 {
		t.Fatalf("find exit %d: %s", code, out)
	}
	for _, want := range []string{"01ABC t3", "01DEF t9", "total 2 mentions in 2 sessions"} {
		if !strings.Contains(out, want) {
			t.Fatalf("find output missing %q: %s", want, out)
		}
	}

	// Engine failure is forwarded, never an empty set.
	bad := filepath.Join(t.TempDir(), "rekal-bad")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\necho 'rekal: Binder Error: column x' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code = runFind(bad, "turtle")
	if code != 2 || !strings.Contains(out, "Binder Error") {
		t.Fatalf("want forwarded engine error exit 2, got code=%d %s", code, out)
	}

	// No args → usage, exit 2. Unknown role → exit 2.
	if _, code = runFind(stub); code != 2 {
		t.Fatalf("no-arg find should exit 2, got %d", code)
	}
	if _, code = runFind(stub, "turtle", "notarole"); code != 2 {
		t.Fatalf("bad role should exit 2, got %d", code)
	}
}
