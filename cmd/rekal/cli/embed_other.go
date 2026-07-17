//go:build !unix

package cli

import (
	"os"
	"syscall"
)

func detachSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func flockExclusiveNB(f *os.File) error {
	_ = f
	return nil
}

func flockUnlock(f *os.File) error {
	_ = f
	return nil
}
