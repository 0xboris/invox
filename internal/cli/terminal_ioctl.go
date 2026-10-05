//go:build linux || darwin

package cli

import (
	"os"
	"syscall"
	"unsafe"
)

// isTerminal reports whether file is a terminal: reading its terminal
// attributes succeeds. Unlike a character-device check, this is false for
// /dev/null.
func isTerminal(file *os.File) bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), ioctlGetTermios, uintptr(unsafe.Pointer(&termios)))
	return errno == 0
}
