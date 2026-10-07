package cmdutil

import (
	"sync"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// Factory holds what commands use to reach outside invox: the standard
// streams, the process environment and the external programs.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Compiler  *tectonic.Compiler
	Editor    *editor.Editor
	Opener    *opener.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer *applemail.Composer

	host func() invoice.Host
}

// NewFactory returns a Factory whose adapters run programs with runner on the
// OS e.GOOS and read the editor settings with e.Getenv.
func NewFactory(ios *iostreams.IOStreams, runner run.Runner, e env.Env) *Factory {
	f := &Factory{
		IOStreams: ios,
		Env:       e,
		host:      sync.OnceValue(func() invoice.Host { return newHost(e) }),
		Compiler:  tectonic.New(runner, ios, e.GOOS),
		Editor:    editor.New(runner, ios, e.GOOS, e.Getenv),
		Opener:    opener.New(runner, ios, e.GOOS),
	}
	if e.GOOS == "darwin" {
		f.Mailer = applemail.New(runner, ios)
	}
	return f
}

// Host returns the user directories, resolved from Env on the first call so
// that a test can set up its environment after building the Factory.
func (f *Factory) Host() invoice.Host {
	return f.host()
}

func newHost(e env.Env) invoice.Host {
	home, err := e.HomeDir()
	if err != nil {
		home = ""
	}
	return invoice.NewHost(invoice.HostInputs{
		GOOS:          e.GOOS,
		Home:          home,
		XDGConfigHome: e.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   e.Getenv("XDG_DATA_HOME"),
		AppData:       e.Getenv("APPDATA"),
	})
}
