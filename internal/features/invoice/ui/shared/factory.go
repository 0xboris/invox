package shared

import (
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/invoice/usecases"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Factory holds what the invoice commands use to reach outside invox: the
// streams, the environment, the user directories and the slice's use cases.
// The composition root builds it.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Hosts     *invoice.HostSource

	LoadInvoice      usecases.LoadInvoice
	RenderTeX        usecases.RenderTeX
	BuildPDF         usecases.BuildPDF
	EditInvoice      usecases.EditInvoice
	ListCustomers    usecases.ListCustomers
	CheckDraftOutput usecases.CheckDraftOutput
	ComposeEmail     usecases.ComposeEmail
	OpenDraft        usecases.OpenDraft
	OpenTempDraft    usecases.OpenTempDraft
}

// Host returns the user directories, resolved on the first call.
func (f *Factory) Host() invoice.Host {
	return f.Hosts.Host()
}
