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
