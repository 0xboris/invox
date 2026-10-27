// Package env holds the parts of the process environment invox depends on:
// the OS, environment variables, the home and working directories, and the
// clock. Main resolves one Env with platform.Env and passes it down, so the
// rest of the code reads these values from what it is given.
package env

import "time"

type Env struct {
	GOOS    string
	Getenv  func(string) string
	HomeDir func() (string, error)
	Getwd   func() (string, error)
	Now     func() time.Time
}
