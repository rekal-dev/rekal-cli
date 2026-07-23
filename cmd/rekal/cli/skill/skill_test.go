package skill

import (
	"strings"
	"testing"
)

// TestAll_UnifiedSkill verifies the collapsed suite: one skill named rekal,
// well-formed tip, and every dispatch target (references + the remaining
// workflow-gate scripts) present. The retrieval/navigation scripts
// (route/view/find/seek/when) moved into the binary as commands.
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

	// The skill must be scriptless for retrieval/navigation — those are commands
	// now. No .py may ship.
	for rel := range s.Files {
		if strings.HasSuffix(rel, ".py") {
			t.Errorf("skill must not ship Python scripts, found %s", rel)
		}
	}

	// Tip must name the commands (progressive disclosure to the binary) and the
	// remaining references/gate scripts.
	for _, needle := range []string{
		"rekal ",
		"rekal find",
		"rekal when",
		"references/ledger.md",
		"references/map.md",
		"references/reference.md",
		"scripts/map.sh",
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
	if !IsScript("scripts/map.sh") {
		t.Fatal("scripts/ should be executable")
	}
	if IsScript("references/ledger.md") {
		t.Fatal("references/ should not be marked script")
	}
}
