//go:build !unix

package db

import "fmt"

func lockIndexPath(path string, exclusive bool) (func(), error) {
	if exclusive {
		return nil, fmt.Errorf("safe index replacement is unsupported on this platform: %s", path)
	}
	return func() {}, nil
}
