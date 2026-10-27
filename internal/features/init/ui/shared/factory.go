// Package shared holds what the init command uses to reach outside invox.
package shared

import (
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Factory holds what the init command uses to reach outside
// invox. The composition root builds it.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Hosts     *invoice.HostSource
}

// Host returns the user directories, resolved on the first call.
func (f *Factory) Host() invoice.Host {
	return f.Hosts.Host()
}
