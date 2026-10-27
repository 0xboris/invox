// Package testsupport holds port implementations that fail, for the error
// paths of use case tests. Only test code imports it.
package testsupport

import (
	"context"

	"github.com/0xboris/invox/internal/domain/ports"
)

// FailingOpener returns Err from every call.
type FailingOpener struct {
	Err error
}

var _ ports.Opener = FailingOpener{}

func (o FailingOpener) Open(ctx context.Context, path string) error {
	return o.Err
}
