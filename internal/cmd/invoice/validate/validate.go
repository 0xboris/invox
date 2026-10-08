// Package validate is the `invox validate` command.
package validate

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

type ValidateOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	InvoicePath   string
	CustomersPath string
	IssuerPath    string
}

// NewCmdValidate returns the validate command. runF replaces validateRun in
// tests.
func NewCmdValidate(f *cmdutil.Factory, runF func(*ValidateOptions) error) *cobra.Command {
	opts := &ValidateOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "validate -i INVOICE.yaml",
		Short: "Validate invoice YAML against customers and issuer data",
		Long: `Validate invoice YAML against customers and issuer data.

Required inputs:
  -i, --input PATH        Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer,
		Example: `$ invox validate -i invoice.yaml
$ invox validate -i invoices/2026-0021.yaml -c customers.yaml -u issuer.yaml
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("validate", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
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
	cmd.ValidArgsFunction = cobra.NoFileCompletions
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
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
	return nil
}
