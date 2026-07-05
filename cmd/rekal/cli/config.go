package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// Config is Rekal's local, per-repo configuration. It lives in the gitignored
// .rekal/config.json — outside the disposable index, so it survives the index
// rebuilds that `index` and `sync` perform. The index content is re-derived,
// but the settings that decide what to derive (and, in future, how to weight
// recall's BM25 / LSA / nomic layers) persist here.
//
// The zero value is the default configuration: no cross-repo import.
type Config struct {
	// LocalImport is the cross-repo local import preference (rekal index
	// --include-all / --include / --no-local). Omitted when unset (default).
	LocalImport localPref `json:"local_import,omitempty"`

	// Future ranking-weight knobs (BM25 / LSA / nomic layer weights) will be
	// added here so recall tuning has a single, durable home.
}

// empty reports whether the config carries no settings — used to remove the
// file entirely rather than leave an empty {} behind.
func (c Config) empty() bool {
	return !c.LocalImport.enabled()
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

// readConfig loads the persisted config. A missing file is the default (zero
// value), not an error.
func readConfig(gitRoot string) (Config, error) {
	data, err := os.ReadFile(configPath(gitRoot))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
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
