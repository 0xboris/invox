// Package cmdutil holds what every command shares: the Factory, and the error
// types commands return so that cli.Main can decide, in one place, what to
// print and which exit code to use.
package cmdutil

import (
	"errors"
	"fmt"
)

// FlagError is a usage error: a bad flag, argument or subcommand. Main points
// to the help of the command that ran. Root points to the root help instead,
// for a global flag or a help topic, which only the root help explains.
type FlagError struct {
	Err  error
	Root bool
}

func (e *FlagError) Error() string { return e.Err.Error() }
func (e *FlagError) Unwrap() error { return e.Err }

// FlagErrorf returns a *FlagError with a formatted message.
func FlagErrorf(format string, args ...any) error {
	return &FlagError{Err: fmt.Errorf(format, args...)}
}

// SilentError means the command failed and has already reported why.
var SilentError = errors.New("SilentError")

// CancelError means the user declined a prompt or interrupted it.
var CancelError = errors.New("CancelError")

// ExecError is the failure of an external program. Code is the program's exit
// code, or -1 when it did not exit normally. invox itself exits 1.
type ExecError struct {
	Program string
	Code    int
	Err     error
}

func (e *ExecError) Error() string {
	if e.Code < 0 {
		return fmt.Sprintf("%s failed: %v", e.Program, e.Err)
	}
	return fmt.Sprintf("%s exited with status %d", e.Program, e.Code)
}
func (e *ExecError) Unwrap() error { return e.Err }
