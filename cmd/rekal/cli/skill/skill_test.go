package skill

import (
	"strings"
	"testing"
)

// TestAll_ShipsSuite verifies the embedded skill suite is present, the base
// "rekal" skill sorts first, and every skill is a well-formed Claude Code skill
// (front-matter with a name matching its directory).
func TestAll_ShipsSuite(t *testing.T) {
	t.Parallel()

	skills := All()
	if len(skills) < 2 {
		t.Fatalf("expected the rekal skill suite, got %d skill(s)", len(skills))
	}

	if skills[0].Name != "rekal" {
		t.Errorf("base skill must sort first, got %q", skills[0].Name)
	}

	want := map[string]bool{
		"rekal":            false,
		"rekal-provenance": false,
		"rekal-reflect":    false,
		"rekal-distill":    false,
	}
	for _, s := range skills {
		if s.Content == "" {
			t.Errorf("%s: empty SKILL.md", s.Name)
		}
		// Every skill must open with YAML front-matter and declare a name that
		// matches its install directory — Claude Code keys skills by that name.
		if !strings.HasPrefix(s.Content, "---\n") {
			t.Errorf("%s: missing front-matter", s.Name)
		}
		if !strings.Contains(s.Content, "name: "+s.Name+"\n") {
			t.Errorf("%s: front-matter name does not match directory", s.Name)
		}
		if !strings.Contains(s.Content, "description:") {
			t.Errorf("%s: front-matter missing description", s.Name)
		}
		if _, ok := want[s.Name]; ok {
			want[s.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("expected skill %q in the suite", name)
		}
	}
}
