package session

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]`)

// SanitizeRepoPath replicates Claude Code's path sanitization:
// non-alphanumeric characters are replaced with dashes.
// e.g. /Users/frank/projects/rekal → -Users-frank-projects-rekal
func SanitizeRepoPath(repoPath string) string {
	return nonAlphanumeric.ReplaceAllString(repoPath, "-")
}

// FindSessionDir returns the Claude Code session directory for the given repo path.
// Returns ~/.claude/projects/<sanitized-repo-path>/.
func FindSessionDir(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sanitized := SanitizeRepoPath(repoPath)
	return filepath.Join(home, ".claude", "projects", sanitized)
}

// FindSessionFiles lists all .jsonl session files in the given directory.
func FindSessionFiles(sessionDir string) ([]string, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(sessionDir, e.Name()))
		}
	}
	return files, nil
}
