package cli

import (
	"fmt"

	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runCustomerList(ios *iostreams.IOStreams, e env.Env, args []string) error {
	spec := customerListSpec()

	opts, _, err := parseCommand(ios, e, spec, args)
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

func runCustomerConfig(ios *iostreams.IOStreams, e env.Env, args []string) error {
	spec := customerConfigSpec()

	opts, _, err := parseCommand(ios, e, spec, args)
	if err != nil {
		return err
	}

	if err := openTextFile(ios, opts.CustomersPath); err != nil {
		return fmt.Errorf("failed to open %s: %w", opts.CustomersPath, err)
	}

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", invoice.DisplayPath(opts.CustomersPath, opts.BaseDir))
	return nil
}
