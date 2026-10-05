package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
)

// canPrompt reports whether the user can answer a confirmation prompt: stdin
// and stderr are both terminals. Tests swap it and restore it with t.Cleanup.
// The IOStreams abstraction (#33) replaces it.
var canPrompt = func() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stderr)
}

// promptInput returns the reader confirmation answers are read from. Tests
// swap it and restore it with t.Cleanup.
var promptInput = func() io.Reader {
	return os.Stdin
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// confirm asks question on stderr and reads a yes/no answer. Anything but
// y or yes, including end of input, is a no.
func confirm(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", question)
	answer, err := bufio.NewReader(promptInput()).ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(os.Stderr)
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
func archiveWithConfirmation(spec commandSpec, opts invoice.Options, errorPrefix string) (invoice.ArchiveResult, int, error) {
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
	if !canPrompt() {
		printCommandError(os.Stderr, spec, fmt.Sprintf(
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
	if !confirm(question) {
		fmt.Fprintf(os.Stderr, "%snot archived; the archive was not changed\n", errorPrefix)
		return invoice.ArchiveResult{}, 2, nil
	}

	archiveOpts.Replace = true
	result, err = invoice.ArchiveInvoice(opts.InvoicePath, archiveOpts)
	return result, 0, err
}

// printArchiveReplacements tells the user on stderr which archived files were
// replaced and where their previous versions are.
func printArchiveReplacements(result invoice.ArchiveResult, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			os.Stderr,
			"Replaced archived invoice %s; previous version kept at %s\n",
			invoice.DisplayPath(backup.Path, baseDir),
			invoice.DisplayPath(backup.BackupPath, baseDir),
		)
	}
}
