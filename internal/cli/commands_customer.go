package cli

import (
	"fmt"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runCustomerList(ios *iostreams.IOStreams, args []string) int {
	spec := customerListSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	customers, err := invoice.ListCustomers(opts.CustomersPath)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	for _, customer := range customers {
		fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", customer.ID, customer.LegalCompanyName, customer.Status)
	}
	return 0
}

func runCustomerConfig(ios *iostreams.IOStreams, args []string) int {
	spec := customerConfigSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	if err := openTextFile(ios, opts.CustomersPath); err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(ios.Out, "Opened %s\n", invoice.DisplayPath(opts.CustomersPath, opts.BaseDir))
	return 0
}
