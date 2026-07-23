//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillVersion_PinAndAutoRefresh covers the version-pinned skill: init
// records the binary version in the installed skill, and a later recall against
// a skill left behind (stale marker) refreshes it in place and re-pins the
// version — without a manual `rekal init`.
func TestSkillVersion_PinAndAutoRefresh(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	marker := filepath.Join(env.RepoDir, ".claude", "skills", "rekal", ".rekal-version")
	skillMD := filepath.Join(env.RepoDir, ".claude", "skills", "rekal", "SKILL.md")

	// init pins a non-empty version.
	pinned, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("init did not pin a skill version marker: %v", err)
	}
	want := strings.TrimSpace(string(pinned))
	if want == "" {
		t.Fatal("skill version marker is empty after init")
	}

	// Simulate a skill left behind by an upgrade: old marker + corrupted skill.
	if err := os.WriteFile(marker, []byte("v0.0.0-stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillMD, []byte("STALE — not the real skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A recall runs the stale-skill check first (SILENCE/empty is fine here).
	_, stderr, _ := env.RunCLI("some recall query")

	if !strings.Contains(stderr, "skill was behind") {
		t.Errorf("expected a refresh notice on stderr, got: %q", stderr)
	}

	// Marker re-pinned to the binary version (not the stale value).
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if g := strings.TrimSpace(string(got)); g != want {
		t.Errorf("marker after refresh = %q, want re-pinned %q", g, want)
	}

	// SKILL.md restored to the real skill (front-matter), not the stale stub.
	md, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(md), "---") {
		t.Errorf("SKILL.md not refreshed; head = %q", firstLine(string(md)))
	}

	// Steady state: a second recall with the marker current does not re-refresh.
	_, stderr2, _ := env.RunCLI("some recall query")
	if strings.Contains(stderr2, "skill was behind") {
		t.Errorf("recall re-refreshed an up-to-date skill: %q", stderr2)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
