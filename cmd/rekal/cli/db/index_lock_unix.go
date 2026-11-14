//go:build unix

package db

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

func lockIndexPath(path string, exclusive bool) (func(), error) {
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}
	deadline := time.Now().Add(openLockRetryBudget)
	for {
		err = syscall.Flock(int(f.Fd()), mode|syscall.LOCK_NB)
		if err == nil {
			return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			break
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(openLockRetryMin)
	}
	_ = f.Close()
	return nil, fmt.Errorf("index replacement lock %s: %w", path, err)
}
