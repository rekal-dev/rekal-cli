//go:build !windows

package nomic

import (
	"os"
	"os/exec"
	"syscall"
)

// setSysProcAttr detaches the daemon process from the parent session.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// lockFile takes an advisory flock on path (created if missing). When block is
// false it returns ok=false immediately if the lock is held elsewhere; when
// true it waits. The returned *os.File must be passed to unlockFile. Used to
// single-flight the daemon (non-blocking) and serialize model-cache extraction
// (blocking) across processes.
func lockFile(path string, block bool) (f *os.File, ok bool, err error) {
	f, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	how := syscall.LOCK_EX
	if !block {
		how |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		_ = f.Close()
		if !block { // held elsewhere
			return nil, false, nil
		}
		return nil, false, err
	}
	return f, true, nil
}

// unlockFile releases a lock taken by lockFile.
func unlockFile(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
