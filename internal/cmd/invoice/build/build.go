// Package build is the `invox build` command.
package build

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// BuildOptions is what build needs: its streams, the use cases and the
// parsed flags.
type BuildOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	InvoicePath   string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	TemplatePath  string
	Archive       bool
	Yes           bool
	DryRun        bool
	Exporter      *cmdutil.Exporter
}

// buildJSON is the --json output of build: the PDF it wrote, the invoice it
// built and, with --archive, where the invoice was archived.
type buildJSON struct {
	Path         string  `json:"path"`
	Input        string  `json:"input"`
	Number       string  `json:"number"`
	CustomerID   string  `json:"customerId"`
	ArchivedPath *string `json:"archivedPath"`
}

// NewCmdBuild returns the build command. runF replaces buildRun in tests.
func NewCmdBuild(f *cmdutil.Factory, runF func(context.Context, *BuildOptions) error) *cobra.Command {
	opts := &BuildOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
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
$ invox build invoice.yaml --archive --dry-run
$ invox build invoice.yaml --json path,number
$ invox build invoices/2026-0021.yaml -o out/2026-0021.pdf -c customers.yaml -u issuer.yaml -t template.tex
`,
		Args: func(cmd *cobra.Command, args []string) error {
			return shared.TakeInput("build", opts.Getwd, &opts.InvoicePath, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput("build", opts.InvoicePath); err != nil {
				return err
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
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Check the invoice and template, print what would be built and archived, and change nothing (does not run Tectonic)")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("output", "pdf")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	_ = cmd.RegisterFlagCompletionFunc("template", cmdutil.CompleteTemplates(f))
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, buildJSON{})
	return cmd
}

func buildRun(ctx context.Context, opts *BuildOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(cmdutil.Files{
		Customers: cmdutil.AbsFlag(baseDir, opts.CustomersPath),
		Issuer:    cmdutil.AbsFlag(baseDir, opts.IssuerPath),
	})
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)
	outputPath := cmdutil.ReplaceExt(invoicePath, ".pdf")
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = cmdutil.AbsPath(baseDir, opts.OutputPath)
	}
	outputDisplay := cmdutil.DisplayPath(outputPath, baseDir)
	invoiceDisplay := cmdutil.DisplayPath(invoicePath, baseDir)

	result, err := svc.Build(ctx, billing.BuildRequest{
		Invoice:  invoicePath,
		Template: opts.TemplatePath,
		Output:   outputPath,
		Archive:  opts.Archive,
		Replace:  opts.Yes,
		Confirm:  shared.ConfirmReplace(ctx, opts.IO, "build", invoicePath, baseDir, fmt.Sprintf("built %s but ", outputDisplay)),
		DryRun:   opts.DryRun,
	})
	var stepErr *billing.StepError
	var execErr *run.ExecError
	switch {
	case errors.As(err, &stepErr) && opts.DryRun:
		return fmt.Errorf("cannot archive %s: %w", invoiceDisplay, stepErr.Err)
	case errors.As(err, &stepErr) && stepErr.Step == billing.StepMark:
		return fmt.Errorf("built %s but failed to update %s: %w", outputDisplay, invoiceDisplay, stepErr.Err)
	case errors.As(err, &stepErr):
		// Main prints only a usage error's inner message and stays silent
		// for CancelError, so the wrap adds nothing to them.
		return fmt.Errorf("built %s but failed to archive %s: %w", outputDisplay, invoiceDisplay, stepErr.Err)
	case errors.As(err, &execErr):
		return &cmdutil.ExecError{Program: "tectonic", Code: execErr.Code, Err: err}
	case err != nil:
		return cmdutil.UsageError("build", err)
	}
	inv := result.Context
	if result.Archived != nil {
		shared.WarnUnread(opts.IO, result.Archived.Unread, baseDir)
	}

	// printResult prints the PDF's path, or with --json the build's result.
	printResult := func(archivedPath *string) error {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, buildJSON{Path: outputPath, Input: invoicePath, Number: inv.InvoiceNumber, CustomerID: inv.CustomerID, ArchivedPath: archivedPath})
		}
		fmt.Fprintln(opts.IO.Out, outputDisplay)
		return nil
	}
	if opts.DryRun {
		fmt.Fprintf(opts.IO.ErrOut, "Would build %s for %s (%s)\n", outputDisplay, inv.CustomerID, inv.InvoiceNumber)
		status := invoice.Status(inv.Header.Status.Trim())
		if next, _ := status.Apply(invoice.Building); next != status {
			fmt.Fprintf(opts.IO.ErrOut, "Would set invoice.status to %s in %s\n", next, invoiceDisplay)
		}
		if result.Archived == nil {
			return printResult(nil)
		}
		shared.PrintArchivePreview(opts.IO, *result.Archived, invoicePath, baseDir)
		return printResult(&result.Archived.Path)
	}
	if result.Archived == nil {
		fmt.Fprintf(opts.IO.ErrOut, "Built %s for %s (%s)\n", outputDisplay, inv.CustomerID, inv.InvoiceNumber)
		return printResult(nil)
	}
	shared.PrintArchiveReplacements(opts.IO, *result.Archived, baseDir)
	fmt.Fprintf(
		opts.IO.ErrOut,
		"Built %s for %s (%s)\nArchived %s -> %s\n",
		outputDisplay,
		inv.CustomerID,
		inv.InvoiceNumber,
		invoiceDisplay,
		cmdutil.DisplayPath(result.Archived.Path, baseDir),
	)
	return printResult(&result.Archived.Path)
}
