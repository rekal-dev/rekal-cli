//go:build windows

package nomic

import (
	"os"
	"os/exec"
)

// setSysProcAttr is a no-op on Windows (Unix sockets not supported).
func setSysProcAttr(_ *exec.Cmd) {}

// lockFile is a no-op on Windows — the cgo daemon (and thus the callers of this
// helper) only builds on unix targets, so this stub exists purely to compile.
func lockFile(_ string, _ bool) (*os.File, bool, error) { return nil, true, nil }

// unlockFile is a no-op on Windows.
func unlockFile(_ *os.File) {}
