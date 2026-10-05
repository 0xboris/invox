// Package cmdutil holds the error types commands return so that cli.Main can
// decide, in one place, what to print and which exit code to use.
package cmdutil

import (
	"errors"
	"fmt"
)

// FlagError is a usage error: a bad flag, argument or subcommand. Command is
// the command whose help explains the usage, for example "customer list", or
// empty for the root command.
type FlagError struct {
	Command string
	Err     error
}

func (e *FlagError) Error() string { return e.Err.Error() }
func (e *FlagError) Unwrap() error { return e.Err }

// FlagErrorf returns a *FlagError for command with a formatted message.
func FlagErrorf(command, format string, args ...any) error {
	return &FlagError{Command: command, Err: fmt.Errorf(format, args...)}
}

// SilentError means the command failed and has already reported why.
var SilentError = errors.New("SilentError")

// CancelError means the user declined a prompt or interrupted it.
var CancelError = errors.New("CancelError")

// ExecError is the failure of an external program that has already reported
// on stderr. invox exits with the program's exit code.
type ExecError struct {
	Program string
	Code    int
	Err     error
}

func (e *ExecError) Error() string { return fmt.Sprintf("%s failed: %v", e.Program, e.Err) }
func (e *ExecError) Unwrap() error { return e.Err }
