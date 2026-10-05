//go:build !linux && !darwin && !windows

package cli

import "os"

// isTerminal reports whether file is a character device. On these systems
// that also counts /dev/null, which then declines at the prompt.
func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
