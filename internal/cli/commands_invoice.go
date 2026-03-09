package cli

import (
	"fmt"
	"os"
	"strings"

	"invox/internal/invoice"
)

func runNew(args []string) int {
	args = reorderArgs(args, map[string]bool{
		"-c":          true,
		"--customers": true,
		"-u":          true,
		"--issuer":    true,
		"-s":          true,
		"--source":    true,
		"-o":          true,
		"--output":    true,
	})

	spec := newSpec()

	opts, extraArgs, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	customerID := strings.TrimSpace(extraArgs[0])
	invoiceNumber, err := invoice.CreateNewInvoice(opts.DefaultsPath, opts.OutputPath, opts.CustomersPath, opts.IssuerPath, customerID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Created %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		customerID,
		invoiceNumber,
	)
	return 0
}

func runIncrement(args []string) int {
	spec := incrementSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	customerID, oldNumber, newNumber, err := invoice.IncrementInvoiceNumber(opts.InvoicePath, opts.CustomersPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Incremented %s for %s: %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		customerID,
		oldNumber,
		newNumber,
	)
	return 0
}

func runValidate(args []string) int {
	spec := validateSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Validation OK: %s for %s, %d line item(s), total %s\n",
		ctx.InvoiceNumber,
		ctx.CustomerID,
		len(ctx.LineItems),
		invoice.FormatCurrency(ctx.TotalCents, ctx.Currency),
	)
	return 0
}

func runRender(args []string) int {
	spec := renderSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := invoice.RenderInvoice(opts.TemplatePath, opts.OutputPath, ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Rendered %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	return 0
}

func runBuild(args []string) int {
	spec := buildSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := invoice.BuildInvoicePDF(opts.TemplatePath, opts.OutputPath, ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Built %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	return 0
}
