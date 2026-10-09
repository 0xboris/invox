// Package render is the `invox render` command.
package render

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
)

type RenderOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	InvoicePath string
	OutputPath  string
	Support     cmdutil.SupportPaths
	DryRun      bool
	Exporter    *cmdutil.Exporter
}

// renderJSON is the --json output of render: the TeX file it wrote and the
// invoice it rendered.
type renderJSON struct {
	Path       string `json:"path"`
	Input      string `json:"input"`
	Number     string `json:"number"`
	CustomerID string `json:"customerId"`
}

// NewCmdRender returns the render command. runF replaces renderRun in tests.
func NewCmdRender(f *cmdutil.Factory, runF func(context.Context, *RenderOptions) error) *cobra.Command {
	opts := &RenderOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	if runF == nil {
		runF = renderRun
	}
	cmd := &cobra.Command{
		Use:   "render [INVOICE.yaml]",
		Short: "Render a LaTeX invoice file from YAML data",
		Long: `Render a LaTeX invoice file from YAML data.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default output:
  invoice.tex in the current directory

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			helptext.LookupTemplate,
		Example: `$ invox render invoice.yaml
$ invox render invoices/2026-0021.yaml -o out/2026-0021.tex -c customers.yaml -u issuer.yaml -t template.tex
$ invox render invoice.yaml --dry-run
$ invox render invoice.yaml --json path
`,
		Args: shared.TakeInput(opts.Getwd, &opts.InvoicePath),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput(opts.InvoicePath); err != nil {
				return err
			}
			if err := shared.RequireExtension(opts.OutputPath, ".tex"); err != nil {
				return err
			}
			return runF(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output TeX path (must end with .tex; default invoice.tex)")
	cmdutil.AddSupportFlags(cmd, f, &opts.Support, billing.CustomersFile, billing.IssuerFile, billing.TemplateFile)
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Check the invoice and template, print the output path and write nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("output", "tex")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, renderJSON{})
	return cmd
}

func renderRun(_ context.Context, opts *RenderOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(opts.Support.Files(baseDir))
	outputPath := filepath.Join(baseDir, "invoice.tex")
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = cmdutil.AbsPath(baseDir, opts.OutputPath)
	}
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)

	ctx, err := svc.Render(billing.RenderRequest{Invoice: invoicePath, Template: opts.Support.Template, Output: outputPath, DryRun: opts.DryRun})
	if err != nil {
		return cmdutil.UsageError(err)
	}
	displayPath := cmdutil.DisplayPath(outputPath, baseDir)
	verb := "Rendered"
	if opts.DryRun {
		verb = "Would render"
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s %s for %s (%s)\n", verb, displayPath, ctx.CustomerID, ctx.InvoiceNumber)
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, renderJSON{Path: outputPath, Input: invoicePath, Number: ctx.InvoiceNumber, CustomerID: ctx.CustomerID})
	}
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
