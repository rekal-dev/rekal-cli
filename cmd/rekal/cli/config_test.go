package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/embedhttp"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

// writeCfgFile writes raw JSON to path, creating parent dirs.
func writeCfgFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMergedConfig_GlobalWithLocalOverride(t *testing.T) {
	// Sets env, so not parallel.
	gitRoot := t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	globalHome := t.TempDir()
	t.Setenv("REKAL_CONFIG_HOME", globalHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	globalPath := filepath.Join(globalHome, "config.json")

	// Global sets the embedding backend and a tuned weight mix.
	writeCfgFile(t, globalPath,
		`{"embedding":{"endpoint":"http://global/v1","model":"gmodel"},`+
			`"weights":{"bm25":0.5,"nomic":0.5},"local_import":{"all":true}}`)

	// 1. global-only: no local file → inherit everything.
	cfg, err := readMergedConfig(gitRoot)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if cfg.Embedding == nil || cfg.Embedding.Endpoint != "http://global/v1" {
		t.Fatalf("global embedding not inherited: %+v", cfg.Embedding)
	}
	w, _ := cfg.Weights.resolve()
	if w.BM25 != 0.5 || w.Nomic != 0.5 {
		t.Fatalf("global weights not inherited: %+v", w)
	}
	// local_import is NOT inherited (guardrail).
	if cfg.LocalImport.enabled() {
		t.Fatalf("local_import must not inherit from global, got %+v", cfg.LocalImport)
	}

	// 2. local overrides one weight and the embedding, inherits the rest.
	writeCfgFile(t, configPath(gitRoot),
		`{"weights":{"bm25":0.9}}`)
	cfg, err = readMergedConfig(gitRoot)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if cfg.Embedding == nil || cfg.Embedding.Endpoint != "http://global/v1" {
		t.Fatalf("embedding should still inherit global: %+v", cfg.Embedding)
	}
	w, _ = cfg.Weights.resolve()
	if w.BM25 != 0.9 { // local override
		t.Fatalf("local bm25 override lost: %v", w.BM25)
	}
	if w.Nomic != 0.5 { // inherited from global
		t.Fatalf("global nomic should be inherited: %v", w.Nomic)
	}

	// 3. local embedding replaces global embedding wholesale.
	writeCfgFile(t, configPath(gitRoot),
		`{"embedding":{"endpoint":"http://local/v1","model":"lmodel"}}`)
	cfg, _ = readMergedConfig(gitRoot)
	if cfg.Embedding.Endpoint != "http://local/v1" {
		t.Fatalf("local embedding should win: %+v", cfg.Embedding)
	}

	// 4. write path (readConfig) is local-only — never carries global values.
	local, err := readConfig(gitRoot)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	if local.Weights != nil {
		t.Fatalf("local-only read must not carry global weights: %+v", local.Weights)
	}
}

func TestMergedConfig_NeitherIsDefaults(t *testing.T) {
	gitRoot := t.TempDir()
	t.Setenv("REKAL_CONFIG_HOME", t.TempDir()) // empty dir, no config.json
	cfg, err := readMergedConfig(gitRoot)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	w, err := cfg.Weights.resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if w != search.DefaultWeights() {
		t.Fatalf("neither global nor local → defaults, got %+v", w)
	}
}

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

func fp(v float64) *float64 { return &v }

func TestWeightsConfig_Resolve(t *testing.T) {
	t.Parallel()

	// Absent config → defaults.
	var absent *weightsConfig
	w, err := absent.resolve()
	if err != nil || w != search.DefaultWeights() {
		t.Fatalf("absent resolve = %+v, %v; want defaults", w, err)
	}

	// Partial override keeps other defaults.
	w, err = (&weightsConfig{BM25: fp(0.5)}).resolve()
	if err != nil {
		t.Fatalf("partial resolve: %v", err)
	}
	if w.BM25 != 0.5 || w.Nomic != search.DefaultWeights().Nomic {
		t.Fatalf("partial resolve = %+v", w)
	}

	// Explicit zero disables a layer (distinct from absent).
	w, err = (&weightsConfig{LSA: fp(0)}).resolve()
	if err != nil || w.LSA != 0 {
		t.Fatalf("lsa=0 resolve = %+v, %v", w, err)
	}

	// Invalid values rejected.
	if _, err := (&weightsConfig{BM25: fp(-1)}).resolve(); err == nil {
		t.Fatal("negative layer weight must be rejected")
	}
	if _, err := (&weightsConfig{SteeringBoost: fp(0)}).resolve(); err == nil {
		t.Fatal("zero steering_boost must be rejected")
	}
	if _, err := (&weightsConfig{SummaryBoost: fp(0)}).resolve(); err == nil {
		t.Fatal("zero summary_boost must be rejected")
	}

	// summary_boost overrides; absent keeps the default.
	w, err = (&weightsConfig{SummaryBoost: fp(2)}).resolve()
	if err != nil || w.SummaryBoost != 2 {
		t.Fatalf("summary_boost resolve = %+v, %v", w, err)
	}
	w, err = (&weightsConfig{BM25: fp(0.5)}).resolve()
	if err != nil || w.SummaryBoost != search.DefaultWeights().SummaryBoost {
		t.Fatalf("absent summary_boost should keep default, got %+v", w)
	}
	if _, err := (&weightsConfig{BM25: fp(0), LSA: fp(0), Nomic: fp(0)}).resolve(); err == nil {
		t.Fatal("all-zero layers must be rejected")
	}
}

func TestEmbeddingConfig_Resolve(t *testing.T) {
	t.Setenv("REKAL_TEST_ENDPOINT", "http://127.0.0.1:9999/v1")
	t.Setenv("REKAL_TEST_KEY", "from-env")

	// Env expansion in endpoint; api_key_env wins over api_key.
	cfg, err := (&embeddingConfig{
		Endpoint:  "$REKAL_TEST_ENDPOINT",
		Model:     "nomic-embed-text-v1.5",
		APIKey:    "hardcoded",
		APIKeyEnv: "REKAL_TEST_KEY",
	}).resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Endpoint != "http://127.0.0.1:9999/v1" {
		t.Fatalf("endpoint = %q, want env-expanded", cfg.Endpoint)
	}
	if cfg.APIKey != "from-env" {
		t.Fatalf("api_key = %q, want the env var to win", cfg.APIKey)
	}
	// Nomic-family model gets the embedded backend's prefixes by default.
	if cfg.QueryPrefix != "search_query: " || cfg.DocumentPrefix != "search_document: " {
		t.Fatalf("nomic prefixes not defaulted: %+v", cfg)
	}

	// Non-nomic model: no implicit prefixes; explicit override respected.
	empty := ""
	cfg, err = (&embeddingConfig{Endpoint: "http://x/v1", Model: "bge-m3", QueryPrefix: &empty}).resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.QueryPrefix != "" || cfg.DocumentPrefix != "" {
		t.Fatalf("non-nomic prefixes should be empty: %+v", cfg)
	}

	// Missing required fields rejected.
	if _, err := (&embeddingConfig{Model: "m"}).resolve(); err == nil {
		t.Fatal("empty endpoint must be rejected")
	}
	if _, err := (&embeddingConfig{Endpoint: "http://x/v1"}).resolve(); err == nil {
		t.Fatal("empty model must be rejected")
	}

	// Default provider is openai.
	cfg, err = (&embeddingConfig{Endpoint: "http://x/v1", Model: "m"}).resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Provider != embedhttp.ProviderOpenAI {
		t.Fatalf("default provider = %q, want %q", cfg.Provider, embedhttp.ProviderOpenAI)
	}

	// Bedrock provider: no text prefixes even for a nomic-shaped name (the
	// asymmetry rides input_type, so prefixes would double-encode).
	cfg, err = (&embeddingConfig{
		Provider: "bedrock",
		Endpoint: "https://bedrock-runtime.us-east-1.amazonaws.com",
		Model:    "cohere.embed-english-v3",
	}).resolve()
	if err != nil {
		t.Fatalf("resolve bedrock: %v", err)
	}
	if cfg.Provider != embedhttp.ProviderBedrock {
		t.Fatalf("provider = %q, want bedrock", cfg.Provider)
	}
	if cfg.QueryPrefix != "" || cfg.DocumentPrefix != "" {
		t.Fatalf("bedrock must not set text prefixes: %+v", cfg)
	}

	// Unknown provider rejected.
	if _, err := (&embeddingConfig{Provider: "azure", Endpoint: "http://x", Model: "m"}).resolve(); err == nil {
		t.Fatal("unknown provider must be rejected")
	}
}
