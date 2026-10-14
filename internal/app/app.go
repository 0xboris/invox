// Package app is the composition root. It builds the adapters, hands each
// slice the parts it needs, and runs the command line.
package app

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/buildinfo"
	"github.com/0xboris/invox/internal/adapters/fsutil"
	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/adapters/programs/editor"
	"github.com/0xboris/invox/internal/adapters/programs/opener"
	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/adapters/programs/tectonic"
	"github.com/0xboris/invox/internal/adapters/starter"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
	cli "github.com/0xboris/invox/internal/features/cli/ui"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Dependencies holds what the commands use to reach outside invox: the
// standard streams, the process environment, the user directories and the
// external programs. A test replaces a field before it calls Main.
type Dependencies struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Hosts     *invoice.HostSource
	Compiler  ports.PDFCompiler
	Editor    ports.Editor
	Opener    ports.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer ports.MailComposer
	// Version is the release version, such as "v1.2.3", or "DEV", and Date
	// its release date (YYYY-MM-DD) or "".
	Version, Date string
}

// NewDependencies returns Dependencies whose adapters run programs with
// runner on the OS e.GOOS and read the user directories and the editor
// settings from e.
func NewDependencies(ios *iostreams.IOStreams, runner run.Runner, e env.Env) *Dependencies {
	return &Dependencies{
		IOStreams: ios,
		Env:       e,
		Hosts:     invoice.NewHostSource(hostResolver(e, fsutil.Files{}, starter.Files)),
		Compiler:  tectonic.New(runner, ios),
		Editor:    editor.New(runner, ios, e.GOOS, e.Getenv),
		Opener:    opener.New(runner, ios, e.GOOS),
		Mailer:    mailComposer(runner, ios, e.GOOS),
		Version:   buildinfo.Version,
		Date:      buildinfo.Date,
	}
}

// Main runs the invox command line args and returns the exit code. SIGINT
// and SIGTERM cancel the command.
func Main(args []string, d *Dependencies) int {
	ctx, stop := platform.SignalContext(context.Background())
	defer stop()
	return mainContext(ctx, args, d)
}

// mainContext runs the invox command line args under ctx and returns the
// exit code.
func mainContext(ctx context.Context, args []string, d *Dependencies) int {
	return cli.Main(ctx, args, cliFactory(d), commands(d)...)
}

// NewRootCmd returns the invox command tree.
func NewRootCmd(d *Dependencies) *cobra.Command {
	return cli.NewRootCmd(cliFactory(d), commands(d)...)
}

// commands returns the commands below the root.
func commands(d *Dependencies) []*cobra.Command {
	var all []*cobra.Command
	all = append(all, invoiceCommands(d)...)
	all = append(all, configCommands(d)...)
	all = append(all, customerCommands(d)...)
	all = append(all, initCommands(d)...)
	all = append(all, templateCommands(d)...)
	all = append(all, versionCommands(d)...)
	return all
}
