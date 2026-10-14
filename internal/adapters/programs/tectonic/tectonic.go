// Package tectonic compiles LaTeX files to PDF with the tectonic program.
package tectonic

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Compiler runs tectonic. Its output goes to stderr, so stdout carries only
// the data a command prints.
type Compiler struct {
	runner run.Runner
	ios    *iostreams.IOStreams
	goos   string
}

var _ ports.PDFCompiler = (*Compiler)(nil)

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

// NotInstalledError is ports.CompilerNotInstalledError.
type NotInstalledError = ports.CompilerNotInstalledError
