package session

import (
	"os"
	"path/filepath"
)

// ProjectsRoot returns the Claude Code projects directory
// (<config>/projects), honoring CLAUDE_CONFIG_DIR the same way FindSessionDir
// does. Returns "" if the home directory can't be resolved.
func ProjectsRoot() string {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".claude")
	}
	return filepath.Join(configDir, "projects")
}

// EnumerateProjectDirs returns every project session directory under the
// Claude Code projects root — one per working directory the agent was ever
// launched in, including non-repo/shell directories. Used by the cross-repo
// local import (rekal index --include-all). Returns nil (no error) when the
// projects root is absent.
func EnumerateProjectDirs() ([]string, error) {
	root := ProjectsRoot()
	if root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs, nil
}

// ProjectDirForRepo returns the project session directory for a specific repo
// path, resolved the same way the current repo is (SanitizeRepoPath under the
// projects root). Used by rekal index --include <repo>.
func ProjectDirForRepo(repoPath string) string {
	root := ProjectsRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, SanitizeRepoPath(repoPath))
}

// DiscoverSessionRefsInDir returns the session refs (trunk transcripts followed
// by their subagent transcripts) in a single Claude Code project directory.
// Unlike ClaudeAdapter.Discover, which resolves the directory from a repo path,
// this takes the project directory directly — the cross-repo import already
// knows the directory it wants to walk.
func DiscoverSessionRefsInDir(projectDir string) ([]SessionRef, error) {
	return discoverSessionRefs(projectDir)
}
