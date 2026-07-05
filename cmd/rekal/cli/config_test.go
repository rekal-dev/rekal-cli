package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_MissingIsDefault(t *testing.T) {
	t.Parallel()
	gitRoot := t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := readConfig(gitRoot)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	if cfg.LocalImport.enabled() {
		t.Fatalf("missing config should be disabled by default, got %+v", cfg)
	}
}

func TestWriteReadConfig_RoundTrip(t *testing.T) {
	t.Parallel()
	gitRoot := t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}

	want := Config{LocalImport: localPref{Repos: []string{"/a/b", "/c/d"}}}
	if err := writeConfig(gitRoot, want); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}

	got, err := readConfig(gitRoot)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	if len(got.LocalImport.Repos) != 2 || got.LocalImport.Repos[0] != "/a/b" || got.LocalImport.Repos[1] != "/c/d" {
		t.Fatalf("round-trip mismatch: got %+v", got.LocalImport)
	}
	if !got.LocalImport.enabled() {
		t.Fatal("expected enabled after writing repos")
	}
}

func TestWriteConfig_EmptyRemovesFile(t *testing.T) {
	t.Parallel()
	gitRoot := t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a non-empty config, then overwrite with an empty one.
	if err := writeConfig(gitRoot, Config{LocalImport: localPref{All: true}}); err != nil {
		t.Fatalf("writeConfig (non-empty): %v", err)
	}
	if _, err := os.Stat(configPath(gitRoot)); err != nil {
		t.Fatalf("config file should exist after non-empty write: %v", err)
	}

	if err := writeConfig(gitRoot, Config{}); err != nil {
		t.Fatalf("writeConfig (empty): %v", err)
	}
	if _, err := os.Stat(configPath(gitRoot)); !os.IsNotExist(err) {
		t.Fatalf("empty config should remove the file, stat err = %v", err)
	}

	// Reading a now-absent file is still the default.
	cfg, err := readConfig(gitRoot)
	if err != nil {
		t.Fatalf("readConfig after removal: %v", err)
	}
	if cfg.LocalImport.enabled() {
		t.Fatal("expected disabled after empty write removes the file")
	}
}

func TestLocalPref_Enabled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		p    localPref
		want bool
	}{
		{"zero", localPref{}, false},
		{"all", localPref{All: true}, true},
		{"repos", localPref{Repos: []string{"/x"}}, true},
		{"empty repos", localPref{Repos: []string{}}, false},
	}
	for _, tc := range cases {
		if got := tc.p.enabled(); got != tc.want {
			t.Errorf("%s: enabled() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestConfigPath(t *testing.T) {
	t.Parallel()
	gitRoot := "/repo"
	want := filepath.Join(gitRoot, ".rekal", "config.json")
	if got := configPath(gitRoot); got != want {
		t.Fatalf("configPath = %q, want %q", got, want)
	}
}
