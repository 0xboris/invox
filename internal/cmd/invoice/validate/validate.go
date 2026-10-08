// Package validate is the `invox validate` command.
package validate

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/money"
)

type ValidateOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	InvoicePath   string
	CustomersPath string
	IssuerPath    string
	Exporter      *cmdutil.Exporter
}

// NewCmdValidate returns the validate command. runF replaces validateRun in
// tests.
func NewCmdValidate(f *cmdutil.Factory, runF func(*ValidateOptions) error) *cobra.Command {
	opts := &ValidateOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
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
		Args: func(cmd *cobra.Command, args []string) error {
			return shared.TakeInput("validate", opts.Getwd, &opts.InvoicePath, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput("validate", opts.InvoicePath); err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return validateRun(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.Flags().StringVarP(&opts.IssuerPath, "issuer", "u", "", "Path to issuer.yaml")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, validationJSON{})
	return cmd
}

func validateRun(opts *ValidateOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "validate", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	issuerPath, err := cmdutil.SupportPath(h, "validate", invoice.Issuer, opts.IssuerPath, baseDir)
	if err != nil {
		return err
	}
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)

	ctx, err := invoice.LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		if problems, ok := invalidInvoiceProblems(err); ok && opts.Exporter != nil {
			if writeErr := opts.Exporter.Write(opts.IO, validationJSON{Errors: problems}); writeErr != nil {
				return writeErr
			}
		}
		return err
	}

	shared.WarnArchivedDuplicate(opts.IO, h, invoicePath, baseDir)

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
