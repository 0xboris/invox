//go:build !windows

package fsutil

import (
	"errors"
	"os"
	"syscall"
)

// syncDir flushes changes to dir's entries, such as a rename into it, to
// disk. Some file systems do not support it; there it does nothing.
func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) {
		return err
	}
	return nil
}
