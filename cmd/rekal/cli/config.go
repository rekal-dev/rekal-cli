package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/embedhttp"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// Config is Rekal's local, per-repo configuration. It lives in the gitignored
// .rekal/config.json — outside the disposable index, so it survives the index
// rebuilds that `index` and `sync` perform. The index content is re-derived,
// but the settings that decide what and how to derive persist here.
//
// The file is never committed (.rekal/ is gitignored by init), so hardcoded
// values stay on this machine; secrets should still prefer env references.
//
// The zero value is the default configuration.
type Config struct {
	// LocalImport is the cross-repo local import preference (rekal index
	// --include-all / --include / --no-local). Omitted when unset (default).
	LocalImport localPref `json:"local_import,omitempty"`

	// Weights tunes recall ranking. Applied at query time only — changing
	// them never requires a reindex. Omitted fields keep their defaults.
	Weights *weightsConfig `json:"weights,omitempty"`

	// Embedding switches deep semantic embeddings from the embedded nomic
	// model to an OpenAI-compatible HTTP endpoint (vLLM, Ollama, TEI, ...).
	// Changing model/endpoint requires a rebuild (rekal index) to regenerate
	// vectors in the new model's space.
	Embedding *embeddingConfig `json:"embedding,omitempty"`
}

// empty reports whether the config carries no settings — used to remove the
// file entirely rather than leave an empty {} behind.
func (c Config) empty() bool {
	return !c.LocalImport.enabled() && c.Weights == nil && c.Embedding == nil
}

// weightsConfig is the JSON shape of the recall-tuning knobs. Pointer fields
// distinguish "absent — keep default" from an explicit 0 (e.g. lsa: 0 turns
// the LSA layer off entirely).
type weightsConfig struct {
	BM25               *float64 `json:"bm25,omitempty"`
	LSA                *float64 `json:"lsa,omitempty"`
	Nomic              *float64 `json:"nomic,omitempty"`
	SteeringBoost      *float64 `json:"steering_boost,omitempty"`
	SummaryBoost       *float64 `json:"summary_boost,omitempty"`
	SubagentDownweight *float64 `json:"subagent_downweight,omitempty"`
}

// resolve merges the config over the defaults and validates. Layer weights
// are ratios (normalized downstream); the boost/downweight multipliers must
// be positive.
func (wc *weightsConfig) resolve() (search.Weights, error) {
	w := search.DefaultWeights()
	if wc == nil {
		return w, nil
	}
	set := func(dst *float64, src *float64, name string, allowZero bool) error {
		if src == nil {
			return nil
		}
		if *src < 0 || (!allowZero && *src == 0) {
			return fmt.Errorf("weights.%s must be %s, got %v", name, map[bool]string{true: ">= 0", false: "> 0"}[allowZero], *src)
		}
		*dst = *src
		return nil
	}
	if err := set(&w.BM25, wc.BM25, "bm25", true); err != nil {
		return w, err
	}
	if err := set(&w.LSA, wc.LSA, "lsa", true); err != nil {
		return w, err
	}
	if err := set(&w.Nomic, wc.Nomic, "nomic", true); err != nil {
		return w, err
	}
	if err := set(&w.SteeringBoost, wc.SteeringBoost, "steering_boost", false); err != nil {
		return w, err
	}
	if err := set(&w.SummaryBoost, wc.SummaryBoost, "summary_boost", false); err != nil {
		return w, err
	}
	if err := set(&w.SubagentDownweight, wc.SubagentDownweight, "subagent_downweight", false); err != nil {
		return w, err
	}
	if w.BM25+w.LSA+w.Nomic <= 0 {
		return w, errors.New("weights: at least one of bm25/lsa/nomic must be > 0")
	}
	return w, nil
}

// embeddingConfig is the JSON shape of the HTTP embedding backend settings.
// Endpoint and api_key expand $VAR/${VAR} references, and api_key_env names
// an env var to read — so secrets can stay in the environment entirely. The
// config file itself is gitignored either way.
type embeddingConfig struct {
	// Provider selects the wire protocol: "openai" (default) for any
	// OpenAI-compatible /embeddings server, or "bedrock" for the Amazon
	// Bedrock runtime (Cohere Embed models, authenticated by a Bedrock API
	// key as the bearer token — no SigV4).
	Provider string `json:"provider,omitempty"`
	// Endpoint is the base URL. OpenAI-compatible: include the version
	// prefix, e.g. "http://127.0.0.1:8000/v1" or "$EMBED_ENDPOINT".
	// Bedrock: the runtime endpoint, e.g.
	// "https://bedrock-runtime.us-east-1.amazonaws.com".
	Endpoint string `json:"endpoint"`
	// Model is sent with each request and keys the stored vectors.
	Model string `json:"model"`
	// APIKey is the bearer token; supports $VAR expansion. Prefer APIKeyEnv.
	APIKey string `json:"api_key,omitempty"`
	// APIKeyEnv names an environment variable holding the bearer token.
	APIKeyEnv string `json:"api_key_env,omitempty"`
	// TimeoutSeconds bounds each request (default embedhttp.DefaultTimeout).
	// Keep small: this runs in the post-commit hook and must never stall a
	// commit — a slow server fails the (non-fatal) embedding step instead.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// QueryPrefix/DocumentPrefix override the task prefixes. When absent and
	// the model name contains "nomic", the nomic prefixes are applied.
	QueryPrefix    *string `json:"query_prefix,omitempty"`
	DocumentPrefix *string `json:"document_prefix,omitempty"`
}

// resolve expands env references and fills defaults.
func (ec *embeddingConfig) resolve() (embedhttp.Config, error) {
	if ec == nil {
		return embedhttp.Config{}, errors.New("embedding config absent")
	}
	provider := ec.Provider
	if provider == "" {
		provider = embedhttp.ProviderOpenAI
	}
	if provider != embedhttp.ProviderOpenAI && provider != embedhttp.ProviderBedrock {
		return embedhttp.Config{}, fmt.Errorf("embedding.provider must be %q or %q, got %q",
			embedhttp.ProviderOpenAI, embedhttp.ProviderBedrock, provider)
	}
	cfg := embedhttp.Config{
		Provider: provider,
		Endpoint: os.ExpandEnv(ec.Endpoint),
		Model:    ec.Model,
		APIKey:   os.ExpandEnv(ec.APIKey),
	}
	if ec.APIKeyEnv != "" {
		if v := os.Getenv(ec.APIKeyEnv); v != "" {
			cfg.APIKey = v
		}
	}
	if cfg.Endpoint == "" {
		return cfg, errors.New("embedding.endpoint is empty (after env expansion)")
	}
	if cfg.Model == "" {
		return cfg, errors.New("embedding.model is required")
	}
	if ec.TimeoutSeconds > 0 {
		cfg.Timeout = time.Duration(ec.TimeoutSeconds) * time.Second
	}

	// Task prefixes apply to the OpenAI wire path only; Bedrock Cohere
	// carries the query/document asymmetry in its input_type field, so
	// prefixes there would double-encode it.
	if provider == embedhttp.ProviderOpenAI {
		// Explicit config wins; nomic-family models default to the prefixes
		// the embedded backend uses, so switching backends keeps the vector
		// space consistent.
		nomicFamily := containsFold(ec.Model, "nomic")
		switch {
		case ec.QueryPrefix != nil:
			cfg.QueryPrefix = *ec.QueryPrefix
		case nomicFamily:
			cfg.QueryPrefix = "search_query: "
		}
		switch {
		case ec.DocumentPrefix != nil:
			cfg.DocumentPrefix = *ec.DocumentPrefix
		case nomicFamily:
			cfg.DocumentPrefix = "search_document: "
		}
	}
	return cfg, nil
}

// containsFold reports whether s contains substr, case-insensitively.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// localPref is the persisted cross-repo import preference. Absent (the zero
// value) means the default: no cross-repo import.
type localPref struct {
	// All imports every local project (rekal index --include-all).
	All bool `json:"all,omitempty"`
	// Repos imports only these repo paths (rekal index --include <repo>).
	Repos []string `json:"repos,omitempty"`
}

// enabled reports whether any cross-repo import is requested.
func (p localPref) enabled() bool {
	return p.All || len(p.Repos) > 0
}

// roots resolves the preference to the project session directories to import.
func (p localPref) roots() ([]string, error) {
	if p.All {
		return session.EnumerateProjectDirs()
	}
	dirs := make([]string, 0, len(p.Repos))
	for _, repo := range p.Repos {
		if dir := session.ProjectDirForRepo(repo); dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

func configPath(gitRoot string) string {
	return filepath.Join(RekalDir(gitRoot), "config.json")
}

// globalConfigPath returns the machine-wide config path (defaults every repo
// inherits), or "" when no home is resolvable. Honors $REKAL_CONFIG_HOME (its
// directory) and $XDG_CONFIG_HOME, else ~/.config/rekal/config.json. Like the
// per-repo file, it is local-only — never committed, pushed, or synced.
func globalConfigPath() string {
	if p := os.Getenv("REKAL_CONFIG_HOME"); p != "" {
		return filepath.Join(p, "config.json")
	}
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" {
		return filepath.Join(p, "rekal", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "rekal", "config.json")
}

// readConfigFile loads one config file. A missing file (or empty path) is the
// default (zero value), not an error.
func readConfigFile(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// readConfig loads the per-repo config only. This is the write-path view: the
// flag handlers read-modify-write it, so it must never carry inherited global
// values (that would bake them into the repo's file). Consumers that want the
// effective settings use readMergedConfig.
func readConfig(gitRoot string) (Config, error) {
	return readConfigFile(configPath(gitRoot))
}

// readMergedConfig loads the machine-wide global config and the per-repo local
// config and merges local over global (see mergeConfig). Used at the
// consumption points — recall weights and the index embedding backend — so a
// backend or tuned weights set once in ~/.config/rekal/config.json apply to
// every repo, while a repo can still override per key.
func readMergedConfig(gitRoot string) (Config, error) {
	global, err := readConfigFile(globalConfigPath())
	if err != nil {
		return Config{}, err
	}
	local, err := readConfigFile(configPath(gitRoot))
	if err != nil {
		return Config{}, err
	}
	return mergeConfig(global, local), nil
}

// mergeConfig resolves the two-tier config: global defaults with per-repo
// override, highest-to-lowest local -> global -> built-in defaults.
//
//   - embedding inherits wholesale — a repo either uses the global backend or
//     replaces the whole block (endpoint/model/key move together).
//   - weights merge field-by-field — a repo can override just bm25 and inherit
//     the rest of the global tuned mix.
//   - local_import is NOT inherited — cross-repo import intent stays per-repo,
//     so a global setting can never silently fold every machine's repos into a
//     project (guardrail against surprise).
func mergeConfig(global, local Config) Config {
	merged := Config{LocalImport: local.LocalImport}
	merged.Embedding = global.Embedding
	if local.Embedding != nil {
		merged.Embedding = local.Embedding
	}
	merged.Weights = mergeWeights(global.Weights, local.Weights)
	return merged
}

// mergeWeights deep-merges the tuning knobs field-by-field: a present local
// field overrides, an absent one inherits global. A nil block on either side
// yields the other unchanged.
func mergeWeights(global, local *weightsConfig) *weightsConfig {
	if local == nil {
		return global
	}
	if global == nil {
		return local
	}
	merged := *global
	if local.BM25 != nil {
		merged.BM25 = local.BM25
	}
	if local.LSA != nil {
		merged.LSA = local.LSA
	}
	if local.Nomic != nil {
		merged.Nomic = local.Nomic
	}
	if local.SteeringBoost != nil {
		merged.SteeringBoost = local.SteeringBoost
	}
	if local.SummaryBoost != nil {
		merged.SummaryBoost = local.SummaryBoost
	}
	if local.SubagentDownweight != nil {
		merged.SubagentDownweight = local.SubagentDownweight
	}
	return &merged
}

// writeConfig persists the config to .rekal/config.json. An empty config
// removes the file so a default setup leaves no residue.
func writeConfig(gitRoot string, c Config) error {
	if c.empty() {
		err := os.Remove(configPath(gitRoot))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(gitRoot), data, 0o644)
}
