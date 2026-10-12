// Package opener opens files in the application the OS associates with them.
package opener

import (
	"context"
	"errors"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

// Opener runs open, xdg-open or `cmd /c start`. Their output goes to stderr,
// so stdout carries only the data a command prints.
type Opener struct {
	runner run.Runner
	ios    *iostreams.IOStreams
	goos   string
}

// New returns an Opener that runs programs with runner on the OS goos.
func New(runner run.Runner, ios *iostreams.IOStreams, goos string) *Opener {
	return &Opener{runner: runner, ios: ios, goos: goos}
}

// Open hands path to the OS. It returns when the opening program exits, which
// can be before the application has read the file. It returns a
// *billing.ToolMissingError when the program is not on PATH, and a
// *billing.ToolFailedError when it fails.
func (o *Opener) Open(ctx context.Context, path string) error {
	cmd := run.Cmd{Stdout: o.ios.ErrOut, Stderr: o.ios.ErrOut}
	switch o.goos {
	case "darwin":
		cmd.Name, cmd.Args = "open", []string{path}
	case "windows":
		cmd.Name, cmd.Args = "cmd", []string{"/c", "start", "", path}
	default:
		cmd.Name, cmd.Args = "xdg-open", []string{path}
	}
	err := o.runner.Run(ctx, cmd)
	if errors.Is(err, run.ErrNotFound) {
		return &billing.ToolMissingError{Tool: cmd.Name}
	}
	var execErr *run.ExecError
	if errors.As(err, &execErr) {
		return &billing.ToolFailedError{Tool: cmd.Name, Code: execErr.Code, Err: err}
	}
	return err
}
