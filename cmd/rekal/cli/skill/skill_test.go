package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ledgerWorkflows are the five answer-type specialist workflows, in the exact
// order the SKILL.md routing gate must list them (ordered, exclusive routing).
var ledgerWorkflows = []string{
	"references/workflows/duration.md",
	"references/workflows/complete-set.md",
	"references/workflows/event-time.md",
	"references/workflows/inference.md",
	"references/workflows/point-fact.md",
}

// rekalSkill returns the single rekal skill.
func rekalSkill(t *testing.T) Skill {
	t.Helper()
	skills := All()
	if len(skills) == 0 || skills[0].Name != "rekal" {
		t.Fatal("rekal skill not found")
	}
	return skills[0]
}

// TestSkill_LedgerWorkflowGate verifies the progressive answer-type routing: the
// five workflow files are embedded, SKILL.md routes to them in order (ordered,
// exclusive), the final-answer contract is present, and the contradictory
// "keep relative phrases relative" advice is gone.
func TestSkill_LedgerWorkflowGate(t *testing.T) {
	t.Parallel()
	s := rekalSkill(t)

	// All five workflow files are embedded.
	for _, w := range ledgerWorkflows {
		if _, ok := s.Files[w]; !ok {
			t.Errorf("missing embedded workflow %s", w)
		}
	}

	// SKILL.md references each workflow exactly once, in the specified order.
	prev := -1
	for _, w := range ledgerWorkflows {
		idx := strings.Index(s.Content, w)
		if idx < 0 {
			t.Errorf("SKILL.md routing gate does not reference %s", w)
			continue
		}
		if strings.Count(s.Content, w) != 1 {
			t.Errorf("SKILL.md references %s %d times, want exactly 1 (exclusive routing)", w, strings.Count(s.Content, w))
		}
		if idx <= prev {
			t.Errorf("SKILL.md lists %s out of order (index %d after %d)", w, idx, prev)
		}
		prev = idx
	}

	// The common verification contract is present.
	if !strings.Contains(s.Content, "### Final answer check") {
		t.Error("SKILL.md missing the '### Final answer check' contract")
	}

	// The contradictory advice must be gone from SKILL.md and every reference.
	for rel, data := range s.Files {
		if strings.Contains(string(data), "keep relative phrases relative") {
			t.Errorf("%s still contains the contradictory phrase 'keep relative phrases relative'", rel)
		}
	}
}

// TestSkill_ContentHashes pins SKILL.md and every reference file to a content
// hash — the shipped, benchmark-measured skill must not drift silently. Update a
// hash here only alongside a deliberate content change to that file.
func TestSkill_ContentHashes(t *testing.T) {
	t.Parallel()
	s := rekalSkill(t)

	want := map[string]string{
		"SKILL.md":                             "c5afd7af6f0fbb122d1edd310f1f938364f23b43e3c35eacba87b8fb692364c6",
		"references/ledger.md":                 "98b3c6a0424772ce31ee6da157ca490a936beec8b88694ff3c602c417d3c72ea",
		"references/setup.md":                  "cd667590919d7725c48f1ca3334cf7c1cf5ec5525b42c96c3e977c6c1143e62b",
		"references/map.md":                    "9434758a67fcded223659227b0e62c02a9c3a8b6a4f9cb005df00fa02ccbc950",
		"references/wiki.md":                   "ce117f95ffd1f0d8d70b3d0c1d3401b641f1ef0ca3beb9a1d3812d4c65bc86a1",
		"references/reference.md":              "bd0a571a8cba25d6a6749e3c97238e9373ebabdf14e748218fdbfd66d0eea58d",
		"references/workflows/duration.md":     "9fc5acd4920405c5a2e0b69d9cdd9efd9d6247e6c31b7fcdc096c8b2d589949f",
		"references/workflows/complete-set.md": "ec228f7937c58847116ec0e3cab9e411a245edd4a442f007dc197fdd20b3322d",
		"references/workflows/event-time.md":   "2e9624e231da7928e348e9e9e5bbc19da1706bd47f18ca233d8e7d21f80b2ed5",
		"references/workflows/inference.md":    "6299b568adeea38007a29862d7cc1ba321c8f89987a299ad96e425a2f9225e9c",
		"references/workflows/point-fact.md":   "673e7071db9195c0f95fb4a08d9f7e4b7d23bb8f8b450c28a251d20fa444f826",
	}
	for rel, wantHash := range want {
		data, ok := s.Files[rel]
		if !ok {
			t.Errorf("missing file %s", rel)
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != wantHash {
			t.Errorf("%s content hash = %s, want %s (update the pin only for a deliberate change)", rel, got, wantHash)
		}
	}
}

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
		"references/setup.md",
		"references/workflows/duration.md",
		"references/workflows/complete-set.md",
		"references/workflows/event-time.md",
		"references/workflows/inference.md",
		"references/workflows/point-fact.md",
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

// pluginRoot is the Claude Code plugin directory, relative to this package.
const pluginRoot = "../../../../plugin"

// TestPlugin_SetupOnly pins the ownership split: the plugin bootstraps (install
// the binary, `rekal init`) and ships exactly one setup skill. The recall skill
// stays embedded in the binary, whose commands it describes. Shipping it in both
// places would load two copies at once, and the plugin copy tracks main while an
// installed binary does not — the newer skill would describe flags the user's
// binary lacks. One owner, no divergence.
func TestPlugin_SetupOnly(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("plugin manifest unreadable: %v", err)
	}
	var manifest struct {
		Name   string   `json:"name"`
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("plugin.json is not valid JSON: %v", err)
	}
	// The name is immutable once published — users install by this slug.
	if manifest.Name != "rekal" {
		t.Errorf("plugin name = %q, want rekal", manifest.Name)
	}
	// SKILL.md sits at the plugin root, so the skill path is the root itself.
	if len(manifest.Skills) != 1 || manifest.Skills[0] != "./" {
		t.Errorf("plugin skills = %v, want [./] (single skill at the plugin root)", manifest.Skills)
	}

	// The plugin's own skill is the setup skill, not the recall skill.
	tip, err := os.ReadFile(filepath.Join(pluginRoot, "SKILL.md"))
	if err != nil {
		t.Fatalf("plugin SKILL.md unreadable: %v", err)
	}
	if !strings.Contains(string(tip), "name: rekal-setup\n") {
		t.Error("plugin SKILL.md must declare name: rekal-setup, distinct from the embedded recall skill")
	}
	if !strings.Contains(string(tip), "rekal init") {
		t.Error("plugin SKILL.md must route the user to rekal init — bootstrapping is its whole job")
	}

	// No recall-skill material may reappear here. A skills/ dir or any of the
	// embedded skill's pages would resurrect the two-copy problem.
	if _, err := os.Stat(filepath.Join(pluginRoot, "skills")); !os.IsNotExist(err) {
		t.Error("plugin must not carry a skills/ directory — the recall skill ships in the binary")
	}
	for _, banned := range []string{"references", "scripts"} {
		if _, err := os.Stat(filepath.Join(pluginRoot, banned)); !os.IsNotExist(err) {
			t.Errorf("plugin must not carry %s/ — that is recall-skill material owned by the binary", banned)
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
