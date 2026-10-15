package app

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/features/invoice/ui/archive"
	buildcmd "github.com/0xboris/invox/internal/features/invoice/ui/build"
	"github.com/0xboris/invox/internal/features/invoice/ui/email"
	"github.com/0xboris/invox/internal/features/invoice/ui/increment"
	newcmd "github.com/0xboris/invox/internal/features/invoice/ui/new"
	"github.com/0xboris/invox/internal/features/invoice/ui/render"
	invoiceshared "github.com/0xboris/invox/internal/features/invoice/ui/shared"
	"github.com/0xboris/invox/internal/features/invoice/ui/validate"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

func invoiceFactory(d *Dependencies) *invoiceshared.Factory {
	return &invoiceshared.Factory{
		IOStreams: d.IOStreams,
		Env:       d.Env,
		Hosts:     d.Hosts,
		Compiler:  d.Compiler,
		Editor:    d.Editor,
		Opener:    d.Opener,
		Mailer:    d.Mailer,
	}
}

func invoiceCommands(d *Dependencies) []*cobra.Command {
	f := invoiceFactory(d)
	return []*cobra.Command{
		newcmd.NewCmdNew(f, nil),
		increment.NewCmdIncrement(f, nil),
		validate.NewCmdValidate(f, nil),
		render.NewCmdRender(f, nil),
		buildcmd.NewCmdBuild(f, nil),
		email.NewCmdEmail(f, nil),
		archive.NewCmdArchive(f, nil),
	}
}

func invoiceTopics(d *Dependencies) []cmdutil.Topic {
	return invoiceshared.Topics(invoiceFactory(d))
}
