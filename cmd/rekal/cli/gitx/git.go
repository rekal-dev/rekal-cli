// Package gitx holds the thin git-plumbing helpers shared by the command and
// transport layers. It sits below both so they can share it without an import
// cycle, and it wraps only os/exec — no rekal domain logic beyond the orphan
// branch name.
package gitx

import (
	"os/exec"
	"strings"
)

// HeadSHA returns the current HEAD commit SHA, or 40 zeros if it can't be read.
func HeadSHA(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return strings.Repeat("0", 40)
	}
	return strings.TrimSpace(string(out))
}

// CurrentBranch returns the current branch name, or "unknown" on error.
func CurrentBranch(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// FilesChanged returns the name-status lines of the HEAD~1..HEAD diff.
func FilesChanged(gitRoot string) []string {
	out, err := exec.Command("git", "-C", gitRoot, "diff", "--name-status", "HEAD~1", "HEAD").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// ShowFile reads a file from a git ref (ref:path). Returns nil if not found.
func ShowFile(gitRoot, ref, path string) []byte {
	out, err := exec.Command("git", "-C", gitRoot, "show", ref+":"+path).Output()
	if err != nil {
		return nil
	}
	return out
}

// ConfigValue reads a git config value, or "" if unset.
func ConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HashObject writes data to the git object store and returns its blob SHA.
func HashObject(gitRoot string, data []byte) (string, error) {
	cmd := exec.Command("git", "-C", gitRoot, "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RekalBranchName returns the per-user orphan branch name, rekal/<email>,
// falling back to rekal/local when no git user.email is configured.
func RekalBranchName() string {
	email := strings.TrimSpace(ConfigValue("user.email"))
	if email == "" {
		email = "local"
	}
	return "rekal/" + email
}
