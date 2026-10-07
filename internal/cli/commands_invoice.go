package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runNew(ios *iostreams.IOStreams, args []string) error {
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

	opts, extraArgs, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	customerID := strings.TrimSpace(extraArgs[0])
	invoiceNumber, outputPath, err := invoice.CreateNewInvoice(opts.DefaultsPath, opts.OutputPath, opts.CustomersPath, opts.IssuerPath, customerID, opts.FromLastInvoice)
	var exists *invoice.OutputExistsError
	if errors.As(err, &exists) {
		return fmt.Errorf("%s; choose a different -o/--output path", exists)
	}
	if err != nil {
		return err
	}
	if opts.EditNewInvoice {
		if err := openTextFile(ios, outputPath); err != nil {
			return fmt.Errorf("created %s but failed to open it: %w", invoice.DisplayPath(outputPath, opts.BaseDir), err)
		}
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Created %s for %s (%s)\n",
		invoice.DisplayPath(outputPath, opts.BaseDir),
		customerID,
		invoiceNumber,
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(outputPath, opts.BaseDir))
	return nil
}

func runIncrement(ios *iostreams.IOStreams, args []string) error {
	spec := incrementSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	customerID, oldNumber, newNumber, err := invoice.IncrementInvoiceNumber(opts.InvoicePath, opts.CustomersPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Incremented %s for %s: %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		customerID,
		oldNumber,
		newNumber,
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(opts.InvoicePath, opts.BaseDir))
	return nil
}

func runValidate(ios *iostreams.IOStreams, args []string) error {
	spec := validateSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		return err
	}

	warnArchivedDuplicate(ios, opts.InvoicePath, opts.BaseDir)

	fmt.Fprintf(
		ios.ErrOut,
		"Validation OK: %s for %s, %d line item(s), total %s\n",
		ctx.InvoiceNumber,
		ctx.CustomerID,
		len(ctx.LineItems),
		formatMoney(ctx.TotalCents, ctx.Currency),
	)
	return nil
}

func runRender(ios *iostreams.IOStreams, args []string) error {
	spec := renderSpec()

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		return err
	}
	if err := invoice.RenderInvoice(opts.TemplatePath, opts.OutputPath, ctx); err != nil {
		return err
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Rendered %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(opts.OutputPath, opts.BaseDir))
	return nil
}

func runEmail(ios *iostreams.IOStreams, args []string) error {
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

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	paths, err := invoice.ResolveEmailDraftPaths(opts.InvoicePath, opts.PDFPath, opts.OutputPath)
	if err != nil {
		return err
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
		return err
	}

	if preferNativeMailCompose && !explicitOutputPath {
		if err := openNativeEmailDraft(ios, emailMessage); err != nil {
			return fmt.Errorf("failed to open editable email draft: %w", err)
		}
	} else if explicitOutputPath {
		if err := writeEmailDraft(opts, paths, paths.OutputPath, opts.OverwriteOutput); err != nil {
			return err
		}
		if err := openDocument(ios, paths.OutputPath); err != nil {
			return fmt.Errorf("created %s but failed to open it: %w", invoice.DisplayPath(paths.OutputPath, opts.BaseDir), err)
		}
	} else {
		draftDir, err := os.MkdirTemp("", "invox-email-*")
		if err != nil {
			return fmt.Errorf("create temporary draft directory: %w", err)
		}
		draftPath := filepath.Join(draftDir, filepath.Base(paths.OutputPath))
		if err := writeEmailDraft(opts, paths, draftPath, false); err != nil {
			_ = os.RemoveAll(draftDir)
			return err
		}
		if err := openDocument(ios, draftPath); err != nil {
			_ = os.RemoveAll(draftDir)
			return fmt.Errorf("failed to open email draft: %w", err)
		}
		if err := cleanupOpenedDocument(draftPath, draftDir); err != nil {
			return fmt.Errorf("opened %s but failed to schedule cleanup: %w", draftPath, err)
		}
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Opened email draft for %s (%s) to %s\n",
		emailMessage.CustomerID,
		emailMessage.InvoiceNumber,
		emailMessage.Recipient,
	)
	if explicitOutputPath {
		fmt.Fprintln(ios.Out, invoice.DisplayPath(paths.OutputPath, opts.BaseDir))
	}
	return nil
}

// writeEmailDraft writes the .eml draft to outputPath.
func writeEmailDraft(opts invoice.Options, paths invoice.EmailDraftPaths, outputPath string, overwrite bool) error {
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
		return fmt.Errorf("%s already exists; pass --force or choose another -o path", invoice.DisplayPath(outputPath, opts.BaseDir))
	}
	return err
}

func runBuild(ios *iostreams.IOStreams, args []string) error {
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

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	ctx, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		return err
	}

	if err := invoice.BuildInvoicePDF(opts.TemplatePath, opts.OutputPath, ctx, invoice.ProcessIO{Stdin: ios.In, Stdout: ios.ErrOut, Stderr: ios.ErrOut}); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &cmdutil.ExecError{Program: "tectonic", Code: exitErr.ExitCode(), Err: err}
		}
		return err
	}
	if err := invoice.MarkInvoiceBuilt(opts.InvoicePath); err != nil {
		return fmt.Errorf(
			"built %s but failed to update %s: %w",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			err,
		)
	}
	if opts.ArchiveAfterBuild {
		errorPrefix := fmt.Sprintf(
			"built %s but ",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		)
		result, err := archiveWithConfirmation(ios, spec, opts, errorPrefix)
		// These already start with errorPrefix, and wrapping a FlagError would
		// repeat "built ... but" in its message.
		if errors.Is(err, cmdutil.CancelError) || errors.As(err, new(*cmdutil.FlagError)) {
			return err
		}
		if err != nil {
			return fmt.Errorf(
				"built %s but failed to archive %s: %w",
				invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
				invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
				err,
			)
		}
		printArchiveReplacements(ios, result, opts.BaseDir)
		fmt.Fprintf(
			ios.ErrOut,
			"Built %s for %s (%s)\nArchived %s -> %s\n",
			invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
			ctx.CustomerID,
			ctx.InvoiceNumber,
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			invoice.DisplayPath(result.Path, opts.BaseDir),
		)
		fmt.Fprintln(ios.Out, invoice.DisplayPath(opts.OutputPath, opts.BaseDir))
		return nil
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Built %s for %s (%s)\n",
		invoice.DisplayPath(opts.OutputPath, opts.BaseDir),
		ctx.CustomerID,
		ctx.InvoiceNumber,
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(opts.OutputPath, opts.BaseDir))
	return nil
}

func runArchive(ios *iostreams.IOStreams, args []string) error {
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

	opts, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	result, err := archiveWithConfirmation(ios, spec, opts, "")
	if err != nil {
		return err
	}

	printArchiveReplacements(ios, result, opts.BaseDir)
	fmt.Fprintf(
		ios.ErrOut,
		"Archived %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		invoice.DisplayPath(result.Path, opts.BaseDir),
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(result.Path, opts.BaseDir))
	return nil
}

func runArchiveEdit(ios *iostreams.IOStreams, args []string) error {
	spec := archiveEditSpec()

	opts, extraArgs, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	outputPath, archivePath, err := invoice.EditArchivedInvoice(strings.TrimSpace(extraArgs[0]), opts.BaseDir)
	if err != nil {
		return err
	}

	fmt.Fprintf(
		ios.ErrOut,
		"Editing %s -> %s\n",
		invoice.DisplayPath(archivePath, opts.BaseDir),
		invoice.DisplayPath(outputPath, opts.BaseDir),
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(outputPath, opts.BaseDir))
	return nil
}

func runArchiveList(ios *iostreams.IOStreams, args []string) error {
	spec := archiveListSpec()

	_, _, err := parseCommand(ios, spec, args)
	if err != nil {
		return err
	}

	archivedInvoices, err := invoice.ListArchivedInvoices()
	if err != nil {
		return err
	}
	archiveDir, err := invoice.ResolveArchiveDir()
	if err != nil {
		return err
	}

	list := table{
		columns:   []column{{header: "FILE"}, {header: "CUSTOMER"}, {header: "ISSUE DATE"}, {header: "STATUS"}},
		emptyHint: "No archived invoices found in " + archiveDir,
	}
	if archiveDir == "" {
		list.emptyHint = "No archived invoices found"
	}
	for _, archivedInvoice := range archivedInvoices {
		list.addRow(archivedInvoice.Filename, archivedInvoice.CustomerID, archivedInvoice.IssueDate, archivedInvoice.Status)
	}
	list.print(ios)
	return nil
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
		"warning: invoice number %s is already used by archived invoice %s; run 'invox increment -i %s' before archiving\n",
		duplicate.InvoiceNumber,
		invoice.DisplayPath(duplicate.ArchivedPath, baseDir),
		invoice.DisplayPath(invoicePath, baseDir),
	)
}
