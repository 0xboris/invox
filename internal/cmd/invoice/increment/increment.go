// Package increment is the `invox increment` command.
package increment

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
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
}

// NewCmdIncrement returns the increment command. runF replaces incrementRun
// in tests.
func NewCmdIncrement(f *cmdutil.Factory, runF func(*IncrementOptions) error) *cobra.Command {
	opts := &IncrementOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "increment",
		Short: "Increment the invoice number in an existing invoice YAML file",
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
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
