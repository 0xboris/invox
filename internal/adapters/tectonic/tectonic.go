// Package tectonic compiles LaTeX files to PDF with the tectonic program.
package tectonic

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
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

// Compile compiles sourcePath and returns the PDF next to it. It implements
// latex.Compiler. It returns a *billing.ToolMissingError when tectonic is not
// on PATH, and a *run.ExecError when tectonic fails.
func (c *Compiler) Compile(ctx context.Context, sourcePath string) (string, error) {
	err := c.runner.Run(ctx, run.Cmd{
		Dir:    filepath.Dir(sourcePath),
		Name:   "tectonic",
		Args:   []string{filepath.Base(sourcePath)},
		Stdin:  c.ios.In,
		Stdout: c.ios.ErrOut,
		Stderr: c.ios.ErrOut,
	})
	if errors.Is(err, run.ErrNotFound) {
		return "", &billing.ToolMissingError{Tool: "tectonic", Hint: installHint(c.goos)}
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".pdf", nil
}

// installHint tells the user how to install tectonic on the OS goos.
func installHint(goos string) string {
	if goos == "darwin" {
		return "Install it with 'brew install tectonic', then rerun this command."
	}
	return "Install it from https://tectonic-typesetting.github.io, then rerun this command."
}
