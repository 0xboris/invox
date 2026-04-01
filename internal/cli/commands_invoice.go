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
		"--from-last": false,
		"-e":          false,
		"--edit":      false,
	})

	spec := newSpec()

	opts, extraArgs, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	customerID := strings.TrimSpace(extraArgs[0])
	invoiceNumber, outputPath, err := invoice.CreateNewInvoice(opts.DefaultsPath, opts.OutputPath, opts.CustomersPath, opts.IssuerPath, customerID, opts.FromLastInvoice)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if opts.EditNewInvoice {
		if err := openTextFile(outputPath); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"created %s but failed to open it: %v\n",
				invoice.DisplayPath(outputPath, opts.BaseDir),
				err,
			)
			return 1
		}
	}

	fmt.Printf(
		"Created %s for %s (%s)\n",
		invoice.DisplayPath(outputPath, opts.BaseDir),
		customerID,
		invoiceNumber,
	)
	return 0
}

func runIncrement(args []string) int {
	args = reorderArgs(args, map[string]bool{
		"-i":          true,
		"--input":     true,
		"-c":          true,
		"--customers": true,
	})

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
	args = reorderArgs(args, map[string]bool{
		"-i":          true,
		"--input":     true,
		"-c":          true,
		"--customers": true,
		"-u":          true,
		"--issuer":    true,
		"--json":      false,
	})

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

	if opts.JSONOutput {
		if err := writeJSON(os.Stdout, validateJSON(ctx)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
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
	args = reorderArgs(args, map[string]bool{
		"-i":          true,
		"--input":     true,
		"-o":          true,
		"--output":    true,
		"-c":          true,
		"--customers": true,
		"-u":          true,
		"--issuer":    true,
		"-t":          true,
		"--template":  true,
	})

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

func runEmail(args []string) int {
	args = reorderArgs(args, map[string]bool{
		"-i":          true,
		"--input":     true,
		"-p":          true,
		"--pdf":       true,
		"-o":          true,
		"--output":    true,
		"-c":          true,
		"--customers": true,
		"-u":          true,
		"--issuer":    true,
		"--to":        true,
		"--subject":   true,
		"--no-open":   false,
	})
	explicitOutputPath := false
	for _, arg := range args {
		if arg == "-o" || arg == "--output" || strings.HasPrefix(arg, "-o=") || strings.HasPrefix(arg, "--output=") {
			explicitOutputPath = true
			break
		}
	}

	spec := emailSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	paths, err := invoice.ResolveEmailDraftPaths(opts.InvoicePath, opts.PDFPath, opts.OutputPath)
	if err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return 2
	}

	if !opts.EmailNoOpen && preferNativeMailCompose && !explicitOutputPath {
		emailMessage, err := invoice.PrepareInvoiceEmail(
			opts.CustomersPath,
			opts.IssuerPath,
			paths.InvoicePath,
			paths.PDFPath,
			opts.EmailTo,
			opts.EmailSubject,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := openNativeEmailDraft(emailMessage); err != nil {
			fmt.Fprintf(os.Stderr, "failed to open editable email draft: %v\n", err)
			return 1
		}
		fmt.Printf(
			"Opened email draft for %s (%s) to %s\n",
			emailMessage.CustomerID,
			emailMessage.InvoiceNumber,
			emailMessage.Recipient,
		)
		return 0
	}

	draft, err := invoice.CreateInvoiceEmailDraft(
		opts.CustomersPath,
		opts.IssuerPath,
		paths.InvoicePath,
		paths.PDFPath,
		paths.OutputPath,
		opts.EmailTo,
		opts.EmailSubject,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if opts.EmailNoOpen {
		fmt.Printf(
			"Created email draft %s for %s (%s) to %s\n",
			invoice.DisplayPath(draft.OutputPath, opts.BaseDir),
			draft.CustomerID,
			draft.InvoiceNumber,
			draft.Recipient,
		)
		return 0
	}

	if err := openDocument(draft.OutputPath); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"created %s but failed to open it: %v\n",
			invoice.DisplayPath(draft.OutputPath, opts.BaseDir),
			err,
		)
		return 1
	}
	if err := cleanupOpenedDocument(draft.OutputPath); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"opened %s but failed to schedule cleanup: %v\n",
			invoice.DisplayPath(draft.OutputPath, opts.BaseDir),
			err,
		)
		return 1
	}

	fmt.Printf(
		"Opened email draft for %s (%s) to %s\n",
		draft.CustomerID,
		draft.InvoiceNumber,
		draft.Recipient,
	)
	return 0
}

func runBuild(args []string) int {
	args = reorderArgs(args, map[string]bool{
		"-i":          true,
		"--input":     true,
		"-o":          true,
		"--output":    true,
		"-c":          true,
		"--customers": true,
		"-u":          true,
		"--issuer":    true,
		"-t":          true,
		"--template":  true,
		"--archive":   false,
	})

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
	if err := invoice.SetInvoiceStatus(opts.InvoicePath, "built"); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"built %s but failed to update %s: %v\n",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			err,
		)
		return 1
	}
	if opts.ArchiveAfterBuild {
		archivePath, err := invoice.ArchiveInvoice(opts.InvoicePath)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"built %s but failed to archive %s: %v\n",
				invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
				invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
				err,
			)
			return 1
		}
		fmt.Printf(
			"Built %s for %s (%s)\nArchived %s -> %s\n",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			ctx.CustomerID,
			ctx.InvoiceNumber,
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			invoice.DisplayPath(archivePath, opts.BaseDir),
		)
		return 0
	}

	fmt.Printf(
		"Built %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	return 0
}

func runArchive(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "edit":
			return runArchiveEdit(args[1:])
		case "list":
			return runArchiveList(args[1:])
		}
	}

	spec := archiveSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	archivePath, err := invoice.ArchiveInvoice(opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Archived %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		invoice.DisplayPath(archivePath, opts.BaseDir),
	)
	return 0
}

func runArchiveEdit(args []string) int {
	spec := archiveEditSpec()

	opts, extraArgs, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	outputPath, archivePath, err := invoice.EditArchivedInvoice(strings.TrimSpace(extraArgs[0]), opts.BaseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(
		"Editing %s -> %s\n",
		invoice.DisplayPath(archivePath, opts.BaseDir),
		invoice.DisplayPath(outputPath, opts.BaseDir),
	)
	return 0
}

func runArchiveList(args []string) int {
	spec := archiveListSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	archivedInvoices, err := invoice.ListArchivedInvoices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if opts.JSONOutput {
		if err := writeJSON(os.Stdout, archiveListJSON(archivedInvoices)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	for _, archivedInvoice := range archivedInvoices {
		fmt.Printf(
			"%s\t%s\t%s\t%s\n",
			archivedInvoice.Filename,
			archivedInvoice.CustomerID,
			archivedInvoice.IssueDate,
			archivedInvoice.Status,
		)
	}
	return 0
}
