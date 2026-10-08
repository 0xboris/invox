// Package increment is the `invox increment` command.
package increment

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

type IncrementOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	InvoicePath   string
	CustomersPath string
	Exporter      *cmdutil.Exporter
}

// incrementJSON is the --json output of increment: the invoice and its old
// and new number.
type incrementJSON struct {
	Path           string `json:"path"`
	Number         string `json:"number"`
	PreviousNumber string `json:"previousNumber"`
	CustomerID     string `json:"customerId"`
}

// NewCmdIncrement returns the increment command. runF replaces incrementRun
// in tests.
func NewCmdIncrement(f *cmdutil.Factory, runF func(*IncrementOptions) error) *cobra.Command {
	opts := &IncrementOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "increment -i INVOICE.yaml",
		Short: "Increment the invoice number in an existing invoice YAML file",
		Long: `Increment the invoice number in an existing invoice YAML file.

Required inputs:
  -i, --input PATH        Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupCustomers,
		Example: `$ invox increment -i invoice.yaml
$ invox increment -i invoices/2026-0022.yaml -c customers.yaml
$ invox increment -i invoice.yaml --json number,previousNumber
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("increment", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput("increment", opts.InvoicePath); err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return incrementRun(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.ValidArgsFunction = cobra.NoFileCompletions
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, incrementJSON{})
	return cmd
}

func incrementRun(opts *IncrementOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "increment", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)

	incremented, err := h.IncrementInvoiceNumber(invoicePath, customersPath)
	if err != nil {
		return err
	}
	shared.WarnSkippedArchiveFiles(opts.IO, incremented.CustomerID, incremented.SkippedArchiveFiles, baseDir)

	displayPath := invoice.DisplayPath(invoicePath, baseDir)
	fmt.Fprintf(opts.IO.ErrOut, "Incremented %s for %s: %s -> %s\n", displayPath, incremented.CustomerID, incremented.OldNumber, incremented.NewNumber)
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, incrementJSON{
			Path:           invoicePath,
			Number:         incremented.NewNumber,
			PreviousNumber: incremented.OldNumber,
			CustomerID:     incremented.CustomerID,
		})
	}
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
