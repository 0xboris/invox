package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// archiveWithConfirmation archives the invoice and, when that replaces
// archived files, asks first unless --yes was passed. errorPrefix starts
// the messages it reports. It returns a *cmdutil.FlagError when there is no
// terminal to ask on, and cmdutil.CancelError when the user declines or ctx is
// cancelled at the prompt.
func archiveWithConfirmation(ctx context.Context, ios *iostreams.IOStreams, h invoice.Host, now func() time.Time, spec commandSpec, opts invoice.Options, errorPrefix string) (invoice.ArchiveResult, error) {
	archiveOpts := invoice.ArchiveOptions{Replace: opts.AssumeYes}
	result, err := h.ArchiveInvoice(now(), opts.InvoicePath, archiveOpts)
	var replaceErr *invoice.ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		return result, err
	}

	paths := make([]string, 0, len(replaceErr.Paths))
	for _, path := range replaceErr.Paths {
		paths = append(paths, invoice.DisplayPath(path, opts.BaseDir))
	}
	replaced := strings.Join(paths, ", ")
	if !ios.CanPrompt() {
		return invoice.ArchiveResult{}, cmdutil.FlagErrorf(
			spec.Name,
			"%sarchiving %s replaces archived invoice %s; pass --yes to replace it (%s)",
			errorPrefix,
			invoice.DisplayPath(opts.InvoicePath, opts.BaseDir),
			replaced,
			cmdutil.WhyNoPrompt(ios),
		)
	}
	question := fmt.Sprintf(
		"Replace archived invoice %s? The previous version is kept in %s.",
		replaced,
		invoice.DisplayPath(replaceErr.HistoryDir, opts.BaseDir),
	)
	confirmed, err := cmdutil.Confirm(ctx, ios, question)
	if err != nil {
		return invoice.ArchiveResult{}, err
	}
	if !confirmed {
		fmt.Fprintf(ios.ErrOut, "%snot archived; the archive was not changed; pass --yes to replace without asking\n", errorPrefix)
		return invoice.ArchiveResult{}, cmdutil.CancelError
	}

	archiveOpts.Replace = true
	return h.ArchiveInvoice(now(), opts.InvoicePath, archiveOpts)
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
