package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/tableprinter"
)

func runArchive(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
	if len(args) > 0 {
		switch args[0] {
		case "edit":
			return runArchiveEdit(f, args[1:])
		case "list":
			return runArchiveList(f, args[1:])
		}
	}

	args = reorderArgs(args, map[string]bool{
		"-i":      true,
		"--input": true,
		"--yes":   false,
	})

	spec := archiveSpec()

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	result, err := shared.ArchiveWithConfirmation(ctx, ios, h, e.Now, spec.Name, opts.InvoicePath, opts.BaseDir, opts.AssumeYes, "")
	if err != nil {
		return err
	}

	shared.PrintArchiveReplacements(ios, result, opts.BaseDir)
	fmt.Fprintf(
		ios.ErrOut,
		"Archived %s -> %s\n",
		invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
		invoice.DisplayPath(result.Path, opts.BaseDir),
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(result.Path, opts.BaseDir))
	return nil
}

func runArchiveEdit(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
	spec := archiveEditSpec()

	opts, extraArgs, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	outputPath, archivePath, err := h.EditArchivedInvoice(strings.TrimSpace(extraArgs[0]), opts.BaseDir)
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

func runArchiveList(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
	spec := archiveListSpec()

	_, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	archivedInvoices, err := h.ListArchivedInvoices()
	if err != nil {
		return err
	}
	archiveDir, err := h.ResolveArchiveDir()
	if err != nil {
		return err
	}

	list := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "FILE"}, {Header: "CUSTOMER"}, {Header: "ISSUE DATE"}, {Header: "STATUS"}},
		EmptyHint: "No archived invoices found in " + archiveDir,
	}
	if archiveDir == "" {
		list.EmptyHint = "No archived invoices found"
	}
	for _, archivedInvoice := range archivedInvoices {
		list.AddRow(archivedInvoice.Filename, archivedInvoice.CustomerID, archivedInvoice.IssueDate, archivedInvoice.Status)
	}
	list.Print(ios)
	return nil
}
