// Package validate is the `invox validate` command.
package validate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/money"
)

type ValidateOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	InvoicePath string
	Support     cmdutil.SupportPaths
	Exporter    *cmdutil.Exporter
}

// NewCmdValidate returns the validate command. runF replaces validateRun in
// tests.
func NewCmdValidate(f *cmdutil.Factory, runF func(context.Context, *ValidateOptions) error) *cobra.Command {
	opts := &ValidateOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	if runF == nil {
		runF = validateRun
	}
	cmd := &cobra.Command{
		Use:   "validate [INVOICE.yaml]",
		Short: "Validate invoice YAML against customers and issuer data",
		Long: `Validate invoice YAML against customers and issuer data.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			`
JSON output:
  --json prints one object, also when the invoice is invalid; invox then
  exits 1. The invoice's fields are null when it is invalid, and errors
  lists each problem with its file, line and field where they are known.
`,
		Example: `$ invox validate invoice.yaml
$ invox validate invoices/2026-0021.yaml -c customers.yaml -u issuer.yaml
$ invox validate invoice.yaml --json valid,total,currency,errors
`,
		Args: shared.TakeInput(opts.Getwd, &opts.InvoicePath),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput(opts.InvoicePath); err != nil {
				return err
			}
			return runF(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmdutil.AddSupportFlags(cmd, f, &opts.Support, billing.CustomersFile, billing.IssuerFile)
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, validationJSON{})
	return cmd
}

func validateRun(_ context.Context, opts *ValidateOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(opts.Support.Files(baseDir))
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)

	result, err := svc.Validate(invoicePath)
	if err != nil {
		err = cmdutil.UsageError(err)
		if problems, ok := invalidInvoiceProblems(err); ok && opts.Exporter != nil {
			if writeErr := opts.Exporter.Write(opts.IO, validationJSON{Errors: problems}); writeErr != nil {
				return writeErr
			}
		}
		return err
	}
	ctx := result.Context

	shared.WarnUnread(opts.IO, result.Unread, baseDir)
	shared.WarnArchivedDuplicate(opts.IO, result.Duplicate, invoicePath, baseDir)

	fmt.Fprintf(
		opts.IO.ErrOut,
		"Validation OK: %s for %s, %d line item(s), total %s\n",
		ctx.InvoiceNumber,
		ctx.CustomerID,
		len(ctx.LineItems),
		formatMoney(ctx.TotalCents, ctx.Currency),
	)
	if opts.Exporter != nil {
		lineItems := len(ctx.LineItems)
		total := money.DecimalString(ctx.TotalCents)
		return opts.Exporter.Write(opts.IO, validationJSON{
			Valid:      true,
			Number:     &ctx.InvoiceNumber,
			CustomerID: &ctx.CustomerID,
			LineItems:  &lineItems,
			Total:      &total,
			Currency:   &ctx.Currency,
			Errors:     []problemJSON{},
		})
	}
	return nil
}
