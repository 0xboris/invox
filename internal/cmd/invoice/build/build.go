// Package build is the `invox build` command.
package build

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// BuildOptions is what build needs: its streams, the compiler, the user
// directories and the parsed flags.
type BuildOptions struct {
	IO       *iostreams.IOStreams
	Compiler *tectonic.Compiler
	Host     func() invoice.Host
	Getwd    func() (string, error)
	Now      func() time.Time

	InvoicePath   string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	TemplatePath  string
	Archive       bool
	Yes           bool
}

// NewCmdBuild returns the build command. runF replaces buildRun in tests.
func NewCmdBuild(f *cmdutil.Factory, runF func(context.Context, *BuildOptions) error) *cobra.Command {
	opts := &BuildOptions{IO: f.IOStreams, Compiler: f.Compiler, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
	cmd := &cobra.Command{
		Use:   "build [INVOICE.yaml]",
		Short: "Render and compile an invoice PDF with Tectonic",
		Long: `Render and compile an invoice PDF with Tectonic.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default output:
  the input path with .pdf extension

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			helptext.LookupTemplate +
			"\n" +
			helptext.ReplacingArchived(true),
		Example: `$ invox build invoice.yaml
$ invox build invoice.yaml --archive
$ invox build invoices/2026-0021.yaml -o out/2026-0021.pdf -c customers.yaml -u issuer.yaml -t template.tex
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if rest := shared.TakeInput(&opts.InvoicePath, args); len(rest) > 0 {
				return cmdutil.FlagErrorf("build", "unexpected arguments: %s", strings.Join(rest, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.InvoicePath) == "" {
				return cmdutil.FlagErrorf("build", "missing required input: INVOICE.yaml or -i, --input")
			}
			if err := shared.RequireExtension("build", opts.OutputPath, ".pdf"); err != nil {
				return err
			}
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return buildRun(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output PDF path (must end with .pdf; default: the input with .pdf)")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.Flags().StringVarP(&opts.IssuerPath, "issuer", "u", "", "Path to issuer.yaml")
	cmd.Flags().StringVarP(&opts.TemplatePath, "template", "t", "", "Template path or name")
	cmd.Flags().BoolVar(&opts.Archive, "archive", false, "Archive the invoice after a successful build")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Replace an archived invoice without asking")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("output", "pdf")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	_ = cmd.RegisterFlagCompletionFunc("template", cmdutil.CompleteTemplates(f))
	return cmd
}

func buildRun(ctx context.Context, opts *BuildOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "build", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	issuerPath, err := cmdutil.SupportPath(h, "build", invoice.Issuer, opts.IssuerPath, baseDir)
	if err != nil {
		return err
	}
	templatePath, err := cmdutil.TemplatePath(h, "build", opts.TemplatePath, baseDir)
	if err != nil {
		return err
	}
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)
	outputPath := shared.ReplaceExt(invoicePath, ".pdf")
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = invoice.AbsPath(baseDir, opts.OutputPath)
	}
	outputDisplay := invoice.DisplayPath(outputPath, baseDir)
	invoiceDisplay := invoice.DisplayPath(invoicePath, baseDir)

	inv, err := invoice.LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		return err
	}
	if err := h.BuildInvoicePDF(ctx, opts.Compiler.Build, templatePath, outputPath, inv); err != nil {
		var execErr *run.ExecError
		if errors.As(err, &execErr) {
			return &cmdutil.ExecError{Program: "tectonic", Code: execErr.Code, Err: err}
		}
		return err
	}
	if err := invoice.MarkInvoiceBuilt(invoicePath); err != nil {
		return fmt.Errorf("built %s but failed to update %s: %w", outputDisplay, invoiceDisplay, err)
	}
	if !opts.Archive {
		fmt.Fprintf(opts.IO.ErrOut, "Built %s for %s (%s)\n", outputDisplay, inv.CustomerID, inv.InvoiceNumber)
		fmt.Fprintln(opts.IO.Out, outputDisplay)
		return nil
	}

	errorPrefix := fmt.Sprintf("built %s but ", outputDisplay)
	result, err := shared.ArchiveWithConfirmation(ctx, opts.IO, h, opts.Now, "build", invoicePath, baseDir, opts.Yes, errorPrefix)
	// Main prints only a usage error's inner message and stays silent for
	// CancelError, so the wrap adds nothing to them.
	if err != nil {
		return fmt.Errorf("built %s but failed to archive %s: %w", outputDisplay, invoiceDisplay, err)
	}
	shared.PrintArchiveReplacements(opts.IO, result, baseDir)
	fmt.Fprintf(
		opts.IO.ErrOut,
		"Built %s for %s (%s)\nArchived %s -> %s\n",
		outputDisplay,
		inv.CustomerID,
		inv.InvoiceNumber,
		invoiceDisplay,
		invoice.DisplayPath(result.Path, baseDir),
	)
	fmt.Fprintln(opts.IO.Out, outputDisplay)
	return nil
}
