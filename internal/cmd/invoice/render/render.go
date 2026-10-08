// Package render is the `invox render` command.
package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

type RenderOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	InvoicePath   string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	TemplatePath  string
	DryRun        bool
	Exporter      *cmdutil.Exporter
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
func NewCmdRender(f *cmdutil.Factory, runF func(*RenderOptions) error) *cobra.Command {
	opts := &RenderOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
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
		Args: func(cmd *cobra.Command, args []string) error {
			return shared.TakeInput("render", &opts.InvoicePath, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput("render", opts.InvoicePath); err != nil {
				return err
			}
			if err := shared.RequireExtension("render", opts.OutputPath, ".tex"); err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return renderRun(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output TeX path (must end with .tex; default invoice.tex)")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.Flags().StringVarP(&opts.IssuerPath, "issuer", "u", "", "Path to issuer.yaml")
	cmd.Flags().StringVarP(&opts.TemplatePath, "template", "t", "", "Template path or name")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Check the invoice and template, print the output path and write nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("output", "tex")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	_ = cmd.RegisterFlagCompletionFunc("template", cmdutil.CompleteTemplates(f))
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, renderJSON{})
	return cmd
}

func renderRun(opts *RenderOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "render", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	issuerPath, err := cmdutil.SupportPath(h, "render", invoice.Issuer, opts.IssuerPath, baseDir)
	if err != nil {
		return err
	}
	templatePath, err := cmdutil.TemplatePath(h, "render", opts.TemplatePath, baseDir)
	if err != nil {
		return err
	}
	outputPath := filepath.Join(baseDir, "invoice.tex")
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = invoice.AbsPath(baseDir, opts.OutputPath)
	}
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)

	ctx, err := invoice.LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		return err
	}
	displayPath := invoice.DisplayPath(outputPath, baseDir)
	verb := "Rendered"
	if opts.DryRun {
		verb = "Would render"
		if _, err := invoice.RenderTeX(templatePath, ctx); err != nil {
			return err
		}
	} else if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s %s for %s (%s)\n", verb, displayPath, ctx.CustomerID, ctx.InvoiceNumber)
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, renderJSON{Path: outputPath, Input: invoicePath, Number: ctx.InvoiceNumber, CustomerID: ctx.CustomerID})
	}
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
