package cmdutil

import (
	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/iostreams"
)

// Factory holds what commands use to reach outside invox: the standard
// streams and the external programs.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Compiler  *tectonic.Compiler
	Editor    *editor.Editor
	Opener    *opener.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer *applemail.Composer
}

// NewFactory returns a Factory whose adapters run programs with runner on the
// OS goos and read the editor settings with getenv.
func NewFactory(ios *iostreams.IOStreams, runner run.Runner, goos string, getenv func(string) string) *Factory {
	f := &Factory{
		IOStreams: ios,
		Compiler:  tectonic.New(runner, ios, goos),
		Editor:    editor.New(runner, ios, goos, getenv),
		Opener:    opener.New(runner, ios, goos),
	}
	if goos == "darwin" {
		f.Mailer = applemail.New(runner, ios)
	}
	return f
}
