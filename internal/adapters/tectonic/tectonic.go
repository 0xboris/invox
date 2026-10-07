// Package tectonic compiles LaTeX files to PDF with the tectonic program.
package tectonic

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/iostreams"
)

// Compiler runs tectonic. Its output goes to stderr, so stdout carries only
// the data a command prints.
type Compiler struct {
	runner run.Runner
	ios    *iostreams.IOStreams
	goos   string
}

// New returns a Compiler that runs tectonic with runner on the OS goos.
func New(runner run.Runner, ios *iostreams.IOStreams, goos string) *Compiler {
	return &Compiler{runner: runner, ios: ios, goos: goos}
}

// Build compiles texPath into a PDF next to it. It returns a
// *NotInstalledError when tectonic is not on PATH, and a *run.ExecError when
// tectonic fails.
func (c *Compiler) Build(ctx context.Context, texPath string) error {
	err := c.runner.Run(ctx, run.Cmd{
		Dir:    filepath.Dir(texPath),
		Name:   "tectonic",
		Args:   []string{filepath.Base(texPath)},
		Stdin:  c.ios.In,
		Stdout: c.ios.ErrOut,
		Stderr: c.ios.ErrOut,
	})
	if errors.Is(err, run.ErrNotFound) {
		return &NotInstalledError{GOOS: c.goos}
	}
	return err
}

// NotInstalledError means tectonic is not on PATH. GOOS selects the install
// hint.
type NotInstalledError struct {
	GOOS string
}

func (e *NotInstalledError) Error() string { return "tectonic not found in PATH" }

// InstallHint tells the user how to install tectonic on their OS.
func (e *NotInstalledError) InstallHint() string {
	if e.GOOS == "darwin" {
		return "Install it with 'brew install tectonic', then rerun this command."
	}
	return "Install it from https://tectonic-typesetting.github.io, then rerun this command."
}
