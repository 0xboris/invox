package cli

import (
	"context"
	"fmt"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/tableprinter"
)

func runCustomerList(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	spec := customerListSpec()

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	customers, err := invoice.ListCustomers(opts.CustomersPath)
	if err != nil {
		return err
	}

	list := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "ID"}, {Header: "NAME", MaxWidth: 40}, {Header: "STATUS"}},
		EmptyHint: "No customers found in " + invoice.DisplayPath(opts.CustomersPath, opts.BaseDir),
	}
	for _, customer := range customers {
		list.AddRow(customer.ID, customer.LegalCompanyName, customer.Status)
	}
	list.Print(ios)
	return nil
}

func runCustomerConfig(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	spec := customerConfigSpec()

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	displayPath := invoice.DisplayPath(opts.CustomersPath, opts.BaseDir)
	if err := cmdutil.OpenInEditor(ctx, f.IOStreams, f.Editor, spec.Name, opts.CustomersPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", opts.CustomersPath, err)
	}

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", displayPath)
	return nil
}
