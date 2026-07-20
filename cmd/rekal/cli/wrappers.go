package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// wrapperMarker identifies shims written by rekal, so clean never deletes a
// user's own file that happens to share the name.
const wrapperMarker = "installed by rekal init"

// wrapperScripts are the skill scripts that get a PATH shim. One shim per
// script, named rekal-<script>.
var wrapperScripts = []string{"route", "view", "find"}

// installPathWrappers writes rekal-route / rekal-view / rekal-find into
// ~/.local/bin. Each shim resolves the *current repo's* installed skill at
// run time (git rev-parse), so one global shim serves every initialized repo
// and the agent skips the $ROOT boilerplate on every pipe. Best-effort: an
// unwritable home degrades to the python3 "$ROOT/…" form the skill also
// documents. No-op on Windows (sh shims are useless there).
func installPathWrappers(errOut io.Writer) {
	if runtime.GOOS == "windows" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		fmt.Fprintf(errOut, "rekal: warning: skill wrappers not installed: %v\n", err)
		return
	}
	for _, name := range wrapperScripts {
		shim := "#!/bin/sh\n# " + wrapperMarker + "; removed by `rekal clean`.\n" +
			"exec python3 \"$(git rev-parse --show-toplevel)/.claude/skills/rekal/scripts/" + name + ".py\" \"$@\"\n"
		path := filepath.Join(binDir, "rekal-"+name)
		if err := os.WriteFile(path, []byte(shim), 0o755); err != nil {
			fmt.Fprintf(errOut, "rekal: warning: write %s: %v\n", path, err)
			return
		}
	}
}

// removePathWrappers deletes the shims installPathWrappers wrote — and only
// those: a file without the marker is someone else's and is left alone.
func removePathWrappers() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, name := range wrapperScripts {
		path := filepath.Join(home, ".local", "bin", "rekal-"+name)
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), wrapperMarker) {
			continue
		}
		_ = os.Remove(path)
	}
}
