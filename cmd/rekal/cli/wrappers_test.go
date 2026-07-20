package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No t.Parallel(): t.Setenv on HOME is process-wide.
func TestPathWrappers_InstallAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	installPathWrappers(os.Stderr)

	binDir := filepath.Join(home, ".local", "bin")
	for _, name := range wrapperScripts {
		path := filepath.Join(binDir, "rekal-"+name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("wrapper %s not installed: %v", name, err)
		}
		if !strings.Contains(string(data), wrapperMarker) {
			t.Fatalf("wrapper %s missing marker", name)
		}
		if !strings.Contains(string(data), "scripts/"+name+".py") {
			t.Fatalf("wrapper %s does not exec its script: %s", name, data)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode()&0o111 == 0 {
			t.Fatalf("wrapper %s not executable: %v %v", name, info, err)
		}
	}

	// A user's own file with a colliding name must survive clean.
	foreign := filepath.Join(binDir, "rekal-route")
	if err := os.WriteFile(foreign, []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	removePathWrappers()

	if _, err := os.Stat(filepath.Join(binDir, "rekal-view")); !os.IsNotExist(err) {
		t.Fatal("rekal-view should be removed")
	}
	if _, err := os.Stat(filepath.Join(binDir, "rekal-find")); !os.IsNotExist(err) {
		t.Fatal("rekal-find should be removed")
	}
	data, err := os.ReadFile(foreign)
	if err != nil || !strings.Contains(string(data), "echo mine") {
		t.Fatal("foreign rekal-route must be left alone")
	}
}
