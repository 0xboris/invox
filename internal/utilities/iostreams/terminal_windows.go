package iostreams

import (
	"os"
	"syscall"
)

// isTerminal reports whether file is a console. Unlike a character-device
// check, this is false for NUL.
func isTerminal(file *os.File) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(file.Fd()), &mode) == nil
}
