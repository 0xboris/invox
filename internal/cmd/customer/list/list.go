// Package list is the `invox customer list` command.
package list

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

type ListOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	CustomersPath string
}

// NewCmdList returns the customer list command. runF replaces listRun in tests.
func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List all customers from customers.yaml",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `List all customers from customers.yaml.

Default lookup:
` +
			helptext.LookupCustomers,
		Example: `$ invox customer list
$ invox customer list -c customers.yaml
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("customer list", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	return cmd
}

func listRun(opts *ListOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	customersPath, err := cmdutil.SupportPath(opts.Host(), "customer list", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}

	customers, err := invoice.ListCustomers(customersPath)
	if err != nil {
		return err
	}

	list := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "ID"}, {Header: "NAME", MaxWidth: 40}, {Header: "STATUS"}},
		EmptyHint: "No customers found in " + invoice.DisplayPath(customersPath, baseDir),
	}
	for _, customer := range customers {
		list.AddRow(customer.ID, customer.LegalCompanyName, customer.Status)
	}
	list.Print(opts.IO)
	return nil
}
