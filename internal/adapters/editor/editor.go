// Package editor opens files in the user's text editor.
package editor

import (
	"context"
	"strings"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/iostreams"
)

// Editor runs $VISUAL, then $EDITOR, then the OS default. The editor's output
// goes to stderr, so stdout carries only the data a command prints.
type Editor struct {
	runner run.Runner
	ios    *iostreams.IOStreams
	goos   string
	getenv func(string) string
}

// New returns an Editor that runs programs with runner on the OS goos and
// reads VISUAL, EDITOR and SHELL with getenv.
func New(runner run.Runner, ios *iostreams.IOStreams, goos string, getenv func(string) string) *Editor {
	return &Editor{runner: runner, ios: ios, goos: goos, getenv: getenv}
}

// Edit opens path in the editor and waits for it to exit.
func (e *Editor) Edit(ctx context.Context, path string) error {
	editor := e.command()
	cmd := run.Cmd{
		Env:    []string{"INVOX_EDITOR=" + editor},
		Stdin:  e.ios.In,
		Stdout: e.ios.ErrOut,
		Stderr: e.ios.ErrOut,
	}
	if e.goos == "windows" {
		cmd.Name = "cmd"
		cmd.Args = []string{"/c", editor, path}
		return e.runner.Run(ctx, cmd)
	}

	cmd.Name = strings.TrimSpace(e.getenv("SHELL"))
	if cmd.Name == "" {
		cmd.Name = "/bin/sh"
	}
	cmd.Args = []string{"-lc", `eval "$INVOX_EDITOR" '"$1"'`, "invox", path}
	return e.runner.Run(ctx, cmd)
}

func (e *Editor) command() string {
	if editor := strings.TrimSpace(e.getenv("VISUAL")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(e.getenv("EDITOR")); editor != "" {
		return editor
	}
	if e.goos == "windows" {
		return "notepad"
	}
	return "vi"
}
