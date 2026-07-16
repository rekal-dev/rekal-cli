package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestMergedConfig_ScoringLineageGlobalOnly(t *testing.T) {
	// Sets env, so not parallel.
	gitRoot := t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	globalHome := t.TempDir()
	t.Setenv("REKAL_CONFIG_HOME", globalHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	writeCfgFile(t, filepath.Join(globalHome, "config.json"),
		`{"scoring_lineage":{"enabled":true,"path":"lineage.jsonl","max_candidates":10}}`)
	// Local tries to disable / override — must be ignored.
	writeCfgFile(t, configPath(gitRoot),
		`{"scoring_lineage":{"enabled":false,"max_candidates":99}}`)

	cfg, err := readMergedConfig(gitRoot)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if cfg.ScoringLineage == nil || !cfg.ScoringLineage.Enabled {
		t.Fatalf("scoring_lineage must come from global: %+v", cfg.ScoringLineage)
	}
	if cfg.ScoringLineage.MaxCandidates != 10 {
		t.Fatalf("local max_candidates must not override global, got %d", cfg.ScoringLineage.MaxCandidates)
	}
	if cfg.ScoringLineage.Path != "lineage.jsonl" {
		t.Fatalf("path = %q, want lineage.jsonl", cfg.ScoringLineage.Path)
	}

	// writeConfig must never persist scoring_lineage into the repo file.
	if err := writeConfig(gitRoot, Config{
		LocalImport:    localPref{All: true},
		ScoringLineage: &scoringLineageConfig{Enabled: true},
	}); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}
	local, err := readConfig(gitRoot)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	if local.ScoringLineage != nil {
		t.Fatalf("writeConfig must strip scoring_lineage, got %+v", local.ScoringLineage)
	}
}

func TestOpenLineage_RotationWritesNDJSON(t *testing.T) {
	// Sets env, so not parallel.
	globalHome := t.TempDir()
	t.Setenv("REKAL_CONFIG_HOME", globalHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg := &scoringLineageConfig{
		Enabled:       true,
		Path:          "scoring-lineage.ndjson",
		MaxCandidates: 5,
		Rotation: &scoringLineageRotation{
			MaxMegabytes: 1,
			MaxBackups:   2,
			MaxAgeDays:   1,
		},
	}
	lin, closer, err := cfg.openLineage(io.Discard)
	if err != nil {
		t.Fatalf("openLineage: %v", err)
	}
	if closer != nil {
		defer closer.Close() //nolint:errcheck
	}
	if lin == nil || lin.RunID() == "" {
		t.Fatal("expected lineage with run_id")
	}
	lin.EmitQuery(search.LineageQuery{Query: "x", Mode: "hybrid"})
	lin.StageResult(search.LineageResult{
		Returned:  []search.LineageReturned{{Rank: 1, SessionID: "s1", Score: 0.5}},
		Counts:    map[string]int{"returned": 1},
		TimingsMS: map[string]int64{"total": 1},
	})
	lin.FlushResult(42)
	if closer != nil {
		_ = closer.Close()
	}
	path := filepath.Join(globalHome, "scoring-lineage.ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), `"event":"query"`) || !strings.Contains(string(data), `"event":"result"`) {
		t.Fatalf("log missing events: %s", data)
	}
	if !strings.Contains(string(data), `"run_id":"`+lin.RunID()+`"`) {
		t.Fatalf("log missing run_id: %s", data)
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

	// facet_boost: absent keeps the shipped default (0.3); explicit values
	// apply; zero is valid (the layer's off switch); negative rejected.
	w, err = (&weightsConfig{BM25: fp(0.5)}).resolve()
	if err != nil || w.FacetBoost != search.DefaultWeights().FacetBoost {
		t.Fatalf("absent facet_boost should keep the default, got %+v, %v", w, err)
	}
	w, err = (&weightsConfig{FacetBoost: fp(0.5)}).resolve()
	if err != nil || w.FacetBoost != 0.5 {
		t.Fatalf("facet_boost resolve = %+v, %v", w, err)
	}
	w, err = (&weightsConfig{FacetBoost: fp(0)}).resolve()
	if err != nil || w.FacetBoost != 0 {
		t.Fatalf("facet_boost=0 must disable the layer, got %+v, %v", w, err)
	}
	if _, err := (&weightsConfig{FacetBoost: fp(-0.1)}).resolve(); err == nil {
		t.Fatal("negative facet_boost must be rejected")
	}
}

// TestMergeWeights_FacetBoost covers the two-tier merge for the facet knob:
// local overrides global, absent local inherits global.
func TestMergeWeights_FacetBoost(t *testing.T) {
	t.Parallel()

	global := &weightsConfig{FacetBoost: fp(0.3)}
	if got := mergeWeights(global, &weightsConfig{}); got.FacetBoost == nil || *got.FacetBoost != 0.3 {
		t.Fatalf("absent local facet_boost should inherit global, got %+v", got.FacetBoost)
	}
	if got := mergeWeights(global, &weightsConfig{FacetBoost: fp(0)}); got.FacetBoost == nil || *got.FacetBoost != 0 {
		t.Fatalf("local facet_boost=0 should override global, got %+v", got.FacetBoost)
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
