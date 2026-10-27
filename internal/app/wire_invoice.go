package app

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/features/invoice/ui/archive"
	"github.com/0xboris/invox/internal/features/invoice/ui/buildcmd"
	"github.com/0xboris/invox/internal/features/invoice/ui/email"
	"github.com/0xboris/invox/internal/features/invoice/ui/increment"
	newcmd "github.com/0xboris/invox/internal/features/invoice/ui/new"
	"github.com/0xboris/invox/internal/features/invoice/ui/render"
	invoiceshared "github.com/0xboris/invox/internal/features/invoice/ui/shared"
	"github.com/0xboris/invox/internal/features/invoice/ui/validate"
	invoiceusecases "github.com/0xboris/invox/internal/features/invoice/usecases"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

func invoiceFactory(d *Dependencies) *invoiceshared.Factory {
	return &invoiceshared.Factory{
		IOStreams: d.IOStreams,
		Env:       d.Env,
		Hosts:     d.Hosts,

		LoadInvoice:      invoiceusecases.NewLoadInvoice(d.Hosts),
		RenderTeX:        invoiceusecases.NewRenderTeX(d.Hosts),
		BuildPDF:         invoiceusecases.NewBuildPDF(d.Hosts, d.Compiler),
		EditInvoice:      invoiceusecases.NewEditInvoice(d.Editor),
		ListCustomers:    invoiceusecases.NewListCustomers(d.Hosts),
		CheckDraftOutput: invoiceusecases.NewCheckDraftOutput(d.Hosts),
		ComposeEmail:     invoiceusecases.NewComposeEmail(d.Mailer),
		OpenDraft:        invoiceusecases.NewOpenDraft(d.Hosts, d.Opener),
		OpenTempDraft:    invoiceusecases.NewOpenTempDraft(d.Hosts, d.Opener),
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
