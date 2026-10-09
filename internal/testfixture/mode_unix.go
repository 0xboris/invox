//go:build !windows

package testfixture

import (
	"io/fs"
	"os"
	"syscall"
)

// SetUmask sets the process umask for the rest of the test. invox reads the
// umask once at start, so this only steers modes it does not set itself.
func SetUmask(t T, umask int) {
	t.Helper()
	old := syscall.Umask(umask)
	t.Cleanup(func() { syscall.Umask(old) })
}

// ProcessUmask returns the umask the test process started with, which invox
// clears from the modes of files and directories that are not private.
func ProcessUmask(t T) fs.FileMode {
	t.Helper()
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	return fs.FileMode(umask)
}

// AssertFileMode fails t unless the permission bits of path are want.
func AssertFileMode(t T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}
