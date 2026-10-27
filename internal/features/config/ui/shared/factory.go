// Package shared holds what the config commands use to reach outside invox.
package shared

import (
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/config/usecases"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Factory holds what the config commands use to reach outside
// invox. The composition root builds it.
type Factory struct {
	IOStreams  *iostreams.IOStreams
	Env        env.Env
	Hosts      *invoice.HostSource
	EditConfig usecases.EditConfig
}

// Host returns the user directories, resolved on the first call.
func (f *Factory) Host() invoice.Host {
	return f.Hosts.Host()
}
