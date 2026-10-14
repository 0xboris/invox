package app

import (
	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/adapters/programs/run"
)

// Live returns the Dependencies of the running process.
func Live() *Dependencies {
	return NewDependencies(platform.Streams(), run.Exec{}, platform.Env())
}
