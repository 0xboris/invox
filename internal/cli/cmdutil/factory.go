package cmdutil

import (
	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/env"
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
}

// NewFactory returns a Factory whose adapters run programs with runner on the
// OS e.GOOS and read the editor settings with e.Getenv.
func NewFactory(ios *iostreams.IOStreams, runner run.Runner, e env.Env) *Factory {
	f := &Factory{
		IOStreams: ios,
		Env:       e,
		Compiler:  tectonic.New(runner, ios, e.GOOS),
		Editor:    editor.New(runner, ios, e.GOOS, e.Getenv),
		Opener:    opener.New(runner, ios, e.GOOS),
	}
	if e.GOOS == "darwin" {
		f.Mailer = applemail.New(runner, ios)
	}
	return f
}
