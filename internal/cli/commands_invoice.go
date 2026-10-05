package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runNew(ios *iostreams.IOStreams, args []string) int {
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

	opts, extraArgs, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	customerID := strings.TrimSpace(extraArgs[0])
	invoiceNumber, outputPath, err := invoice.CreateNewInvoice(opts.DefaultsPath, opts.OutputPath, opts.CustomersPath, opts.IssuerPath, customerID, opts.FromLastInvoice)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}
	if opts.EditNewInvoice {
		if err := openTextFile(ios, outputPath); err != nil {
			fmt.Fprintf(
				ios.ErrOut,
				"created %s but failed to open it: %v\n",
				invoice.DisplayPath(outputPath, opts.BaseDir),
				err,
			)
			return 1
		}
	}

	fmt.Fprintf(
		ios.Out,
		"Created %s for %s (%s)\n",
		invoice.DisplayPath(outputPath, opts.BaseDir),
		customerID,
		invoiceNumber,
	)
	return 0
}

func runIncrement(ios *iostreams.IOStreams, args []string) int {
	spec := incrementSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	customerID, oldNumber, newNumber, err := invoice.IncrementInvoiceNumber(opts.InvoicePath, opts.CustomersPath)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(
		ios.Out,
		"Incremented %s for %s: %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		customerID,
		oldNumber,
		newNumber,
	)
	return 0
}

func runValidate(ios *iostreams.IOStreams, args []string) int {
	spec := validateSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	warnArchivedDuplicate(ios, opts.InvoicePath, opts.BaseDir)

	fmt.Fprintf(
		ios.Out,
		"Validation OK: %s for %s, %d line item(s), total %s\n",
		ctx.InvoiceNumber,
		ctx.CustomerID,
		len(ctx.LineItems),
		invoice.FormatCurrency(ctx.TotalCents, ctx.Currency),
	)
	return 0
}

func runRender(ios *iostreams.IOStreams, args []string) int {
	spec := renderSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}
	if err := invoice.RenderInvoice(opts.TemplatePath, opts.OutputPath, ctx); err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(
		ios.Out,
		"Rendered %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	return 0
}

func runEmail(ios *iostreams.IOStreams, args []string) int {
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
		"--force":     false,
	})
	explicitOutputPath := false
	for _, arg := range args {
		if arg == "-o" || arg == "--output" || strings.HasPrefix(arg, "-o=") || strings.HasPrefix(arg, "--output=") {
			explicitOutputPath = true
			break
		}
	}

	spec := emailSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	paths, err := invoice.ResolveEmailDraftPaths(opts.InvoicePath, opts.PDFPath, opts.OutputPath)
	if err != nil {
		printCommandError(ios.ErrOut, spec, err.Error())
		return 2
	}

	emailMessage, err := invoice.PrepareInvoiceEmail(
		opts.CustomersPath,
		opts.IssuerPath,
		paths.InvoicePath,
		paths.PDFPath,
		opts.EmailTo,
		opts.EmailSubject,
	)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	if preferNativeMailCompose && !explicitOutputPath {
		if err := openNativeEmailDraft(ios, emailMessage); err != nil {
			fmt.Fprintf(ios.ErrOut, "failed to open editable email draft: %v\n", err)
			return 1
		}
	} else if explicitOutputPath {
		if code := writeEmailDraft(ios, opts, paths, paths.OutputPath, opts.OverwriteOutput); code != 0 {
			return code
		}
		if err := openDocument(ios, paths.OutputPath); err != nil {
			fmt.Fprintf(
				ios.ErrOut,
				"created %s but failed to open it: %v\n",
				invoice.DisplayPath(paths.OutputPath, opts.BaseDir),
				err,
			)
			return 1
		}
	} else {
		draftDir, err := os.MkdirTemp("", "invox-email-*")
		if err != nil {
			fmt.Fprintf(ios.ErrOut, "create temporary draft directory: %v\n", err)
			return 1
		}
		draftPath := filepath.Join(draftDir, filepath.Base(paths.OutputPath))
		if code := writeEmailDraft(ios, opts, paths, draftPath, false); code != 0 {
			_ = os.RemoveAll(draftDir)
			return code
		}
		if err := openDocument(ios, draftPath); err != nil {
			_ = os.RemoveAll(draftDir)
			fmt.Fprintf(ios.ErrOut, "failed to open email draft: %v\n", err)
			return 1
		}
		if err := cleanupOpenedDocument(draftPath, draftDir); err != nil {
			fmt.Fprintf(
				ios.ErrOut,
				"opened %s but failed to schedule cleanup: %v\n",
				draftPath,
				err,
			)
			return 1
		}
	}

	fmt.Fprintf(
		ios.Out,
		"Opened email draft for %s (%s) to %s\n",
		emailMessage.CustomerID,
		emailMessage.InvoiceNumber,
		emailMessage.Recipient,
	)
	return 0
}

// writeEmailDraft writes the .eml draft to outputPath. It returns a non-zero exit code
// after reporting a failure on stderr.
func writeEmailDraft(ios *iostreams.IOStreams, opts invoice.Options, paths invoice.EmailDraftPaths, outputPath string, overwrite bool) int {
	_, err := invoice.CreateInvoiceEmailDraft(
		opts.CustomersPath,
		opts.IssuerPath,
		paths.InvoicePath,
		paths.PDFPath,
		outputPath,
		overwrite,
		opts.EmailTo,
		opts.EmailSubject,
	)
	if errors.Is(err, fs.ErrExist) {
		fmt.Fprintf(
			ios.ErrOut,
			"%s already exists; pass --force or choose another -o path\n",
			invoice.DisplayPath(outputPath, opts.BaseDir),
		)
		return 1
	}
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}
	return 0
}

func runBuild(ios *iostreams.IOStreams, args []string) int {
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
		"--yes":       false,
	})

	spec := buildSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	if err := invoice.BuildInvoicePDF(opts.TemplatePath, opts.OutputPath, ctx, invoice.ProcessIO{Stdin: ios.In, Stdout: ios.Out, Stderr: ios.ErrOut}); err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}
	if err := invoice.MarkInvoiceBuilt(opts.InvoicePath); err != nil {
		fmt.Fprintf(
			ios.ErrOut,
			"built %s but failed to update %s: %v\n",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			err,
		)
		return 1
	}
	if opts.ArchiveAfterBuild {
		errorPrefix := fmt.Sprintf(
			"built %s but ",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		)
		result, exitCode, err := archiveWithConfirmation(ios, spec, opts, errorPrefix)
		if exitCode != 0 {
			return exitCode
		}
		if err != nil {
			fmt.Fprintf(
				ios.ErrOut,
				"built %s but failed to archive %s: %v\n",
				invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
				invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
				err,
			)
			printDuplicateNumberHint(ios, err, opts.BaseDir)
			return 1
		}
		printArchiveReplacements(ios, result, opts.BaseDir)
		fmt.Fprintf(
			ios.Out,
			"Built %s for %s (%s)\nArchived %s -> %s\n",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			ctx.CustomerID,
			ctx.InvoiceNumber,
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			invoice.DisplayPath(result.Path, opts.BaseDir),
		)
		return 0
	}

	fmt.Fprintf(
		ios.Out,
		"Built %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	return 0
}

func runArchive(ios *iostreams.IOStreams, args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "edit":
			return runArchiveEdit(ios, args[1:])
		case "list":
			return runArchiveList(ios, args[1:])
		}
	}

	args = reorderArgs(args, map[string]bool{
		"-i":      true,
		"--input": true,
		"--yes":   false,
	})

	spec := archiveSpec()

	opts, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	result, exitCode, err := archiveWithConfirmation(ios, spec, opts, "")
	if exitCode != 0 {
		return exitCode
	}
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		printDuplicateNumberHint(ios, err, opts.BaseDir)
		return 1
	}

	printArchiveReplacements(ios, result, opts.BaseDir)
	fmt.Fprintf(
		ios.Out,
		"Archived %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		invoice.DisplayPath(result.Path, opts.BaseDir),
	)
	return 0
}

func runArchiveEdit(ios *iostreams.IOStreams, args []string) int {
	spec := archiveEditSpec()

	opts, extraArgs, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	outputPath, archivePath, err := invoice.EditArchivedInvoice(strings.TrimSpace(extraArgs[0]), opts.BaseDir)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(
		ios.Out,
		"Editing %s -> %s\n",
		invoice.DisplayPath(archivePath, opts.BaseDir),
		invoice.DisplayPath(outputPath, opts.BaseDir),
	)
	return 0
}

func runArchiveList(ios *iostreams.IOStreams, args []string) int {
	spec := archiveListSpec()

	_, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	archivedInvoices, err := invoice.ListArchivedInvoices()
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	list := table{
		columns:   []column{{header: "FILE"}, {header: "CUSTOMER"}, {header: "ISSUE DATE"}, {header: "STATUS"}},
		emptyHint: "No archived invoices found",
	}
	for _, archivedInvoice := range archivedInvoices {
		list.addRow(archivedInvoice.Filename, archivedInvoice.CustomerID, archivedInvoice.IssueDate, archivedInvoice.Status)
	}
	list.print(ios)
	return 0
}

// printDuplicateNumberHint tells the user how to resolve an archive refusal
// caused by an invoice number that is already archived.
func printDuplicateNumberHint(ios *iostreams.IOStreams, err error, baseDir string) {
	var duplicate *invoice.DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		return
	}
	fmt.Fprintf(
		ios.ErrOut,
		"Run `invox increment -i %s` to give it the next free number, then archive it again.\n",
		invoice.DisplayPath(duplicate.InvoicePath, baseDir),
	)
}

// warnArchivedDuplicate warns on stderr when the invoice's number is already
// used by an archived invoice. It never fails validation.
func warnArchivedDuplicate(ios *iostreams.IOStreams, invoicePath, baseDir string) {
	err := invoice.CheckArchivedNumberUnique(invoicePath)
	if err == nil {
		return
	}
	var duplicate *invoice.DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		fmt.Fprintf(ios.ErrOut, "warning: could not check the archive for duplicate invoice numbers: %v\n", err)
		return
	}
	fmt.Fprintf(
		ios.ErrOut,
		"warning: invoice number %s is already used by archived invoice %s; run `invox increment -i %s` before archiving\n",
		duplicate.InvoiceNumber,
		invoice.DisplayPath(duplicate.ArchivedPath, baseDir),
		invoice.DisplayPath(invoicePath, baseDir),
	)
}
