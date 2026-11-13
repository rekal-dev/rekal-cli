package transport

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
)

// CommitFiles is a PROTOTYPE writer: preserve the existing tree, update only
// supplied files, and compare-and-swap the ref. It is not wired into Rekal.
// parentOverride exists only to exercise a stale concurrent writer in the proof.
func CommitFiles(gitRoot, ref, parentOverride string, updates map[string][]byte) (string, error) {
	parent := parentOverride
	if parent == "" {
		out, err := exec.Command("git", "-C", gitRoot, "rev-parse", ref).Output()
		if err != nil { return "", err }
		parent = strings.TrimSpace(string(out))
	}
	tmp, err := os.CreateTemp("", "rekal-memory-proof-index-*")
	if err != nil { return "", err }
	index := tmp.Name()
	if err := tmp.Close(); err != nil { return "", err }
	if err := os.Remove(index); err != nil { return "", err }
	defer os.Remove(index)
	defer os.Remove(index + ".lock")
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", gitRoot}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE=" + index)
		out, err := cmd.CombinedOutput()
		if err != nil { return "", fmt.Errorf("git %s: %w: %s", args[0], err, out) }
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := run("read-tree", parent); err != nil { return "", err }
	paths := make([]string, 0, len(updates))
	for path := range updates { paths = append(paths, path) }
	sort.Strings(paths)
	for _, path := range paths {
		blob, err := gitx.HashObject(gitRoot, updates[path])
		if err != nil { return "", err }
		if _, err := run("update-index", "--add", "--cacheinfo", "100644," + blob + "," + path); err != nil { return "", err }
	}
	tree, err := run("write-tree")
	if err != nil { return "", err }
	commit, err := run("commit-tree", tree, "-p", parent, "-m", "rekal: synthetic memory proof")
	if err != nil { return "", err }
	if _, err := run("update-ref", ref, commit, parent); err != nil { return "", err }
	return commit, nil
}
