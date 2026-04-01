package cli

import (
	"fmt"
	"os"

	"invox/internal/invoice"
)

func runCustomerList(args []string) int {
	spec := customerListSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	customers, err := invoice.ListCustomers(opts.CustomersPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if opts.JSONOutput {
		if err := writeJSON(os.Stdout, customerListJSON(customers)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	for _, customer := range customers {
		fmt.Printf("%s\t%s\t%s\n", customer.ID, customer.LegalCompanyName, customer.Status)
	}
	return 0
}

func runCustomerConfig(args []string) int {
	spec := customerConfigSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	if err := openTextFile(opts.CustomersPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Opened %s\n", invoice.DisplayPath(opts.CustomersPath, opts.BaseDir))
	return 0
}
