// Package platform reads the running process: its environment, its standard
// streams and the signals it receives. Main calls it once and passes the
// results down, so the rest of the code never touches the process.
package platform

import (
	"os"
	"runtime"
	"time"

	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Env returns the Env of the running process.
func Env() env.Env {
	return env.Env{
		GOOS:    runtime.GOOS,
		Getenv:  os.Getenv,
		HomeDir: os.UserHomeDir,
		Getwd:   os.Getwd,
		Now:     time.Now,
	}
}

// Streams returns the process streams. INVOX_FORCE_TTY set to anything but
// the empty string makes stdout count as a terminal.
func Streams() *iostreams.IOStreams {
	return newStreams(os.Stdin, os.Stdout, os.Stderr)
}

func newStreams(in, out, errOut *os.File) *iostreams.IOStreams {
	return iostreams.New(in, out, errOut, isTerminal(in), os.Getenv("INVOX_FORCE_TTY") != "" || isTerminal(out), isTerminal(errOut))
}
