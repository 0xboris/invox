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
}

// NewCmdRender returns the render command. runF replaces renderRun in tests.
func NewCmdRender(f *cmdutil.Factory, runF func(*RenderOptions) error) *cobra.Command {
	opts := &RenderOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "render -i INVOICE.yaml",
		Short: "Render a LaTeX invoice file from YAML data",
		Long: `Render a LaTeX invoice file from YAML data.

Required inputs:
  -i, --input PATH        Path to the invoice YAML file

Default output:
  invoice.tex in the current directory

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			helptext.LookupTemplate,
		Example: `$ invox render -i invoice.yaml
$ invox render -i invoices/2026-0021.yaml -o out/2026-0021.tex -c customers.yaml -u issuer.yaml -t template.tex
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("render", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
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
	cmd.ValidArgsFunction = cobra.NoFileCompletions
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("output", "tex")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	_ = cmd.RegisterFlagCompletionFunc("template", cmdutil.CompleteTemplates(f))
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
		return err
	}

	displayPath := invoice.DisplayPath(outputPath, baseDir)
	fmt.Fprintf(opts.IO.ErrOut, "Rendered %s for %s (%s)\n", displayPath, ctx.CustomerID, ctx.InvoiceNumber)
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
