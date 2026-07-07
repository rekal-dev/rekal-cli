package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
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

func TestApplyLocalPrefFlags(t *testing.T) {
	t.Parallel()

	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.SetErr(io.Discard)
		return cmd
	}

	t.Run("mutually exclusive", func(t *testing.T) {
		t.Parallel()
		gitRoot := t.TempDir()
		if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := applyLocalPrefFlags(newCmd(), gitRoot, true, nil, true); err == nil {
			t.Fatal("expected error for --include-all with --no-local")
		}
	})

	t.Run("no flags leaves preference untouched", func(t *testing.T) {
		t.Parallel()
		gitRoot := t.TempDir()
		if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeConfig(gitRoot, Config{LocalImport: localPref{All: true}}); err != nil {
			t.Fatal(err)
		}
		if err := applyLocalPrefFlags(newCmd(), gitRoot, false, nil, false); err != nil {
			t.Fatalf("applyLocalPrefFlags: %v", err)
		}
		cfg, err := readConfig(gitRoot)
		if err != nil {
			t.Fatal(err)
		}
		if !cfg.LocalImport.All {
			t.Fatal("plain rebuild must honor, not clear, the remembered preference")
		}
	})

	t.Run("include-all persists, no-local clears", func(t *testing.T) {
		t.Parallel()
		gitRoot := t.TempDir()
		if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := applyLocalPrefFlags(newCmd(), gitRoot, true, nil, false); err != nil {
			t.Fatalf("--include-all: %v", err)
		}
		cfg, err := readConfig(gitRoot)
		if err != nil {
			t.Fatal(err)
		}
		if !cfg.LocalImport.All {
			t.Fatal("--include-all did not persist")
		}

		if err := applyLocalPrefFlags(newCmd(), gitRoot, false, nil, true); err != nil {
			t.Fatalf("--no-local: %v", err)
		}
		cfg, err = readConfig(gitRoot)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.LocalImport.enabled() {
			t.Fatal("--no-local did not clear the preference")
		}
	})

	t.Run("include normalizes to absolute paths", func(t *testing.T) {
		gitRoot := t.TempDir() // no t.Parallel: relies on process cwd
		if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := applyLocalPrefFlags(newCmd(), gitRoot, false, []string{"rel/path"}, false); err != nil {
			t.Fatalf("--include: %v", err)
		}
		cfg, err := readConfig(gitRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.LocalImport.Repos) != 1 || !filepath.IsAbs(cfg.LocalImport.Repos[0]) {
			t.Fatalf("repos = %v, want one absolute path", cfg.LocalImport.Repos)
		}
	})
}

func TestConfigPath(t *testing.T) {
	t.Parallel()
	gitRoot := "/repo"
	want := filepath.Join(gitRoot, ".rekal", "config.json")
	if got := configPath(gitRoot); got != want {
		t.Fatalf("configPath = %q, want %q", got, want)
	}
}
