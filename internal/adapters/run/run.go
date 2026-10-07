// Package run starts external programs.
package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Cmd is one invocation of an external program.
type Cmd struct {
	Dir  string
	Name string
	Args []string
	// Env is added to invox's own environment.
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Runner runs a Cmd and waits for it to exit.
type Runner interface {
	Run(ctx context.Context, cmd Cmd) error
}

// ErrNotFound means the program is not on PATH.
var ErrNotFound = exec.ErrNotFound

// ExecError means the program ran and exited with a non-zero Code, or with -1
// when a signal ended it. Name is the program, for callers that name it in
// their message, as the editor error in #41 will.
type ExecError struct {
	Name string
	Code int
	Err  error
}

func (e *ExecError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

func (e *ExecError) Unwrap() error { return e.Err }

// Exec runs programs with os/exec.
type Exec struct{}

func (Exec) Run(ctx context.Context, c Cmd) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExecError{Name: c.Name, Code: exitErr.ExitCode(), Err: err}
	}
	return err
}
