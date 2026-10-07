package cli

import (
	"context"
	"fmt"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runCustomerList(ios *iostreams.IOStreams, args []string) error {
	spec := customerListSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	customers, err := invoice.ListCustomers(opts.CustomersPath)
	if err != nil {
		return err
	}

	list := table{
		columns:   []column{{header: "ID"}, {header: "NAME", maxWidth: 40}, {header: "STATUS"}},
		emptyHint: "No customers found in " + invoice.DisplayPath(opts.CustomersPath, opts.BaseDir),
	}
	for _, customer := range customers {
		list.addRow(customer.ID, customer.LegalCompanyName, customer.Status)
	}
	list.print(ios)
	return nil
}

func runCustomerConfig(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	spec := customerConfigSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	if err := f.Editor.Edit(ctx, opts.CustomersPath); err != nil {
		return fmt.Errorf("failed to open %s: %w", opts.CustomersPath, err)
	}

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", invoice.DisplayPath(opts.CustomersPath, opts.BaseDir))
	return nil
}
