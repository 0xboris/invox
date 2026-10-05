package cli

import (
	"fmt"

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

	for _, customer := range customers {
		fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", customer.ID, customer.LegalCompanyName, customer.Status)
	}
	return nil
}

func runCustomerConfig(ios *iostreams.IOStreams, args []string) error {
	spec := customerConfigSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	if err := openTextFile(ios, opts.CustomersPath); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "Opened %s\n", invoice.DisplayPath(opts.CustomersPath, opts.BaseDir))
	return nil
}
