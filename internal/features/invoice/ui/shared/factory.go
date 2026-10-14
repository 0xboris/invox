package shared

import (
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Factory holds what the invoice commands use to reach outside
// invox. The composition root builds it.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Hosts     *invoice.HostSource
	Compiler  ports.PDFCompiler
	Editor    ports.Editor
	Opener    ports.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer ports.MailComposer
}

// Host returns the user directories, resolved on the first call.
func (f *Factory) Host() invoice.Host {
	return f.Hosts.Host()
}
