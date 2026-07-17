//go:build unix

package cli

import (
	"fmt"
	"os"
	"syscall"
)

func detachSysProcAttr() *syscall.SysProcAttr {
	// New session so the child survives after index/sync exits.
	return &syscall.SysProcAttr{Setsid: true}
}

func flockExclusiveNB(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("lock busy: %w", err)
	}
	return nil
}

func flockUnlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
