//go:build unix

package fsutil

import (
	"io/fs"
	"syscall"
)

// umask is read once at package init: reading it clears it for a moment, and
// a file another goroutine created in that moment would ignore it.
var umask = readUmask()

func readUmask() fs.FileMode {
	mask := syscall.Umask(0)
	syscall.Umask(mask)
	return fs.FileMode(mask)
}
