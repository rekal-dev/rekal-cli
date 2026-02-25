package versioncheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// CheckAndNotify performs a version check and notifies the user if a newer version is available.
// Silent on all errors to avoid interrupting CLI operations.
func CheckAndNotify(w io.Writer, currentVersion string) {
	if currentVersion == "dev" || currentVersion == "" {
		return
	}

	if err := ensureGlobalConfigDir(); err != nil {
		return
	}

	cache, err := loadCache()
	if err != nil {
		cache = &VersionCache{}
	}

	if time.Since(cache.LastCheckTime) < checkInterval {
		return
	}

	latestVersion, err := fetchLatestVersion()

	cache.LastCheckTime = time.Now()
	_ = saveCache(cache)

	if err != nil {
		return
	}

	if isOutdated(currentVersion, latestVersion) {
		printNotification(w, currentVersion, latestVersion)
	}
}

func globalConfigDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, globalConfigDirName), nil
}

func ensureGlobalConfigDir() error {
	configDir, err := globalConfigDirPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}

func cacheFilePath() (string, error) {
	configDir, err := globalConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, cacheFileName), nil
}

func loadCache() (*VersionCache, error) {
	filePath, err := cacheFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}
	var cache VersionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}
	return &cache, nil
}

func saveCache(cache *VersionCache) error {
	filePath, err := cacheFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}
	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, ".version_check_tmp_")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing cache: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpFile.Name(), filePath); err != nil {
		return fmt.Errorf("renaming cache file: %w", err)
	}
	return nil
}

func fetchLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "rekal-cli")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching release info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	version, err := parseGitHubRelease(body)
	if err != nil {
		return "", fmt.Errorf("parsing release: %w", err)
	}

	return version, nil
}

func parseGitHubRelease(body []byte) (string, error) {
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}
	if release.Prerelease {
		return "", errors.New("only prerelease versions available")
	}
	if release.TagName == "" {
		return "", errors.New("empty tag name")
	}
	return release.TagName, nil
}

func isOutdated(current, latest string) bool {
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	return semver.Compare(current, latest) < 0
}

func updateCommand() string {
	execPath, err := os.Executable()
	if err != nil {
		return "curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash"
	}

	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	if strings.Contains(realPath, "/Cellar/") || strings.Contains(realPath, "/homebrew/") {
		return "brew upgrade rekal"
	}

	return "curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash"
}

func printNotification(w io.Writer, current, latest string) {
	msg := fmt.Sprintf("\nA newer version of Rekal CLI is available: %s (current: %s)\nRun '%s' to update.\n",
		latest, current, updateCommand())
	_, _ = fmt.Fprint(w, msg)
}
