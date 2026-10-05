package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// confirm asks question on stderr and reads a yes/no answer. Anything but
// y or yes, including end of input, is a no.
func confirm(ios *iostreams.IOStreams, question string) bool {
	fmt.Fprintf(ios.ErrOut, "%s [y/N] ", question)
	answer, err := bufio.NewReader(ios.In).ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(ios.ErrOut)
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// archiveWithConfirmation archives the invoice and, when that replaces
// archived files, asks first unless --yes was passed. errorPrefix starts
// every message it prints. A non-zero exit code means it stopped and has
// already reported why; err is a failure the caller reports.
func archiveWithConfirmation(ios *iostreams.IOStreams, spec commandSpec, opts invoice.Options, errorPrefix string) (invoice.ArchiveResult, int, error) {
	archiveOpts := invoice.ArchiveOptions{Replace: opts.AssumeYes}
	result, err := invoice.ArchiveInvoice(opts.InvoicePath, archiveOpts)
	var replaceErr *invoice.ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		return result, 0, err
	}

	paths := make([]string, 0, len(replaceErr.Paths))
	for _, path := range replaceErr.Paths {
		paths = append(paths, invoice.DisplayPath(path, opts.BaseDir))
	}
	replaced := strings.Join(paths, ", ")
	if !ios.CanPrompt() {
		printCommandError(ios.ErrOut, spec, fmt.Sprintf(
			"%sarchiving %s replaces archived invoice %s; pass --yes to replace it (no terminal to ask on)",
			errorPrefix,
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			replaced,
		))
		return invoice.ArchiveResult{}, 2, nil
	}
	question := fmt.Sprintf(
		"Replace archived invoice %s? The previous version is kept in %s.",
		replaced,
		invoice.DisplayPath(replaceErr.HistoryDir, opts.BaseDir),
	)
	if !confirm(ios, question) {
		fmt.Fprintf(ios.ErrOut, "%snot archived; the archive was not changed; pass --yes to replace without asking\n", errorPrefix)
		return invoice.ArchiveResult{}, 2, nil
	}

	archiveOpts.Replace = true
	result, err = invoice.ArchiveInvoice(opts.InvoicePath, archiveOpts)
	return result, 0, err
}

// printArchiveReplacements tells the user on stderr which archived files were
// replaced and where their previous versions are.
func printArchiveReplacements(ios *iostreams.IOStreams, result invoice.ArchiveResult, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Replaced archived invoice %s; previous version kept at %s\n",
			invoice.DisplayPath(backup.Path, baseDir),
			invoice.DisplayPath(backup.BackupPath, baseDir),
		)
	}
}
