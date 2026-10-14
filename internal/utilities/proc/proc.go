// Package proc holds the error a failed external program returns, so that
// every layer can tell one apart without knowing how programs are started.
package proc

import "fmt"

// ExitError means a program ran and exited with a non-zero Code, or with -1
// when a signal ended it. Name is the program.
type ExitError struct {
	Name string
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }
