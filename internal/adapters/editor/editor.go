// Package editor opens files in the user's text editor.
package editor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
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
// reads VISUAL and EDITOR with getenv.
func New(runner run.Runner, ios *iostreams.IOStreams, goos string, getenv func(string) string) *Editor {
	return &Editor{runner: runner, ios: ios, goos: goos, getenv: getenv}
}

// Edit opens path in the editor and waits for it to exit. When the editor
// fails it returns a *billing.ToolFailedError that names the editor setting,
// such as editor "code -w".
func (e *Editor) Edit(ctx context.Context, path string) error {
	editor := e.command()
	cmd, err := e.invocation(editor, path)
	if err != nil {
		return err
	}
	cmd.Stdin = e.ios.In
	cmd.Stdout = e.ios.ErrOut
	cmd.Stderr = e.ios.ErrOut

	err = e.runner.Run(ctx, cmd)
	var execErr *run.ExecError
	if errors.As(err, &execErr) {
		return &billing.ToolFailedError{Tool: fmt.Sprintf("editor %q", editor), Code: execErr.Code, Err: err}
	}
	return err
}

// invocation returns the program and arguments that open path with editor.
// On Unix a setting with shell syntax runs through sh -c, never as a login
// shell, so no profile prints into the terminal; "$@" passes path as an
// argument rather than as shell text. Windows has no sh, so the setting is
// always split and run directly.
func (e *Editor) invocation(editor, path string) (run.Cmd, error) {
	posix := e.goos != "windows"
	if posix && needsShell(editor) {
		return run.Cmd{Name: "/bin/sh", Args: []string{"-c", editor + ` "$@"`, "sh", path}}, nil
	}
	words, err := splitWords(editor, posix)
	if err != nil {
		return run.Cmd{}, fmt.Errorf("cannot parse editor %q: %w", editor, err)
	}
	if len(words) == 0 || words[0] == "" {
		return run.Cmd{}, fmt.Errorf("cannot parse editor %q: no program to run", editor)
	}
	return run.Cmd{Name: words[0], Args: append(words[1:], path)}, nil
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
