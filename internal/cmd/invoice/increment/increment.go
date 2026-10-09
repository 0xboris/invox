// Package increment is the `invox increment` command.
package increment

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
)

type IncrementOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	InvoicePath string
	Support     cmdutil.SupportPaths
	DryRun      bool
	Exporter    *cmdutil.Exporter
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
	opts := &IncrementOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "increment [INVOICE.yaml]",
		Short: "Increment the invoice number in an existing invoice YAML file",
		Long: `Increment the invoice number in an existing invoice YAML file.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupCustomers,
		Example: `$ invox increment invoice.yaml
$ invox increment invoices/2026-0022.yaml -c customers.yaml
$ invox increment invoice.yaml --dry-run
$ invox increment invoice.yaml --json number,previousNumber
`,
		Args: shared.TakeInput(opts.Getwd, &opts.InvoicePath),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput(opts.InvoicePath); err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return incrementRun(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmdutil.AddSupportFlags(cmd, f, &opts.Support, billing.CustomersFile)
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the old and new number and change nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, incrementJSON{})
	return cmd
}

func incrementRun(opts *IncrementOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(opts.Support.Files(baseDir))
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)

	incremented, err := svc.Increment(invoicePath, opts.DryRun)
	if err != nil {
		return cmdutil.UsageError(err)
	}
	shared.WarnUnread(opts.IO, incremented.Unread, baseDir)
	shared.WarnSkippedArchiveFiles(opts.IO, incremented.CustomerID, incremented.Skipped, baseDir)

	displayPath := cmdutil.DisplayPath(invoicePath, baseDir)
	verb := "Incremented"
	if opts.DryRun {
		verb = "Would increment"
	}
	fmt.Fprintf(opts.IO.ErrOut, "%s %s for %s: %s -> %s\n", verb, displayPath, incremented.CustomerID, incremented.OldNumber, incremented.NewNumber)
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
