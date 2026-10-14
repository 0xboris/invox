package app

import (
	"github.com/0xboris/invox/internal/adapters/programs/applemail"
	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// mailComposer returns Apple Mail on macOS and nil elsewhere.
func mailComposer(runner run.Runner, ios *iostreams.IOStreams, goos string) ports.MailComposer {
	if goos != "darwin" {
		return nil
	}
	return applemail.New(runner, ios)
}
