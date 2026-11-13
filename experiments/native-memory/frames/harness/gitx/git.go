package gitx

import (
	"os/exec"
	"strings"
)

func ConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func HashObject(gitRoot string, data []byte) (string, error) {
	cmd := exec.Command("git", "-C", gitRoot, "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func RekalBranchName() string {
	email := strings.TrimSpace(ConfigValue("user.email"))
	if email == "" {
		email = "local"
	}
	return "rekal/" + email
}
