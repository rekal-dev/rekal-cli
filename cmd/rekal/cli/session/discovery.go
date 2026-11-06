package session

import (
	"path/filepath"
	"strings"
)

// RepoMatcher lets checkpoint cache expensive transcript-to-repo discovery.
// read returns all recorded working directories; an empty result is cacheable.
type RepoMatcher func(path, repoPath string, read func(string) ([]string, error)) bool

// CachedDiscoverer is optional: adapters with repo-scoped directories need no
// global transcript scan. The cache is owned by checkpoint, not the adapter.
type CachedDiscoverer interface {
	DiscoverCached(repoPath string, match RepoMatcher) ([]SessionRef, error)
}

// RepoPathsMatch accepts an exact working directory or one inside the repo.
func RepoPathsMatch(paths []string, repoPath string) bool {
	for _, cwd := range paths {
		rel, err := filepath.Rel(repoPath, cwd)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func uncachedRepoMatch(path, repoPath string, read func(string) ([]string, error)) bool {
	paths, err := read(path)
	return err == nil && RepoPathsMatch(paths, repoPath)
}
