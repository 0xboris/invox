package shared

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

// ArchiveWithConfirmation archives the invoice at invoicePath and, when that replaces
// archived files, asks first unless assumeYes (--yes) is set. errorPrefix starts
// the messages it reports. It returns a *cmdutil.FlagError when there is no
// terminal to ask on, and cmdutil.CancelError when the user declines or ctx is
// cancelled at the prompt.
func ArchiveWithConfirmation(ctx context.Context, ios *iostreams.IOStreams, h invoice.Host, now func() time.Time, command, invoicePath, baseDir string, assumeYes bool, errorPrefix string) (invoice.ArchiveResult, error) {
	archiveOpts := invoice.ArchiveOptions{Replace: assumeYes}
	result, err := h.ArchiveInvoice(now(), invoicePath, archiveOpts)
	var replaceErr *invoice.ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		return result, err
	}

	paths := make([]string, 0, len(replaceErr.Paths))
	for _, path := range replaceErr.Paths {
		paths = append(paths, invoice.DisplayPath(path, baseDir))
	}
	replaced := strings.Join(paths, ", ")
	if !ios.CanPrompt() {
		return invoice.ArchiveResult{}, cmdutil.FlagErrorf(
			command,
			"%sarchiving %s replaces archived invoice %s; pass --yes to replace it (%s)",
			errorPrefix,
			invoice.DisplayPath(invoicePath, baseDir),
			replaced,
			cmdutil.WhyNoPrompt(ios),
		)
	}
	question := fmt.Sprintf(
		"Replace archived invoice %s? The previous version is kept in %s.",
		replaced,
		invoice.DisplayPath(replaceErr.HistoryDir, baseDir),
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
	return h.ArchiveInvoice(now(), invoicePath, archiveOpts)
}

// PrintArchiveReplacements tells the user on stderr which archived files were
// replaced and where their previous versions are.
func PrintArchiveReplacements(ios *iostreams.IOStreams, result invoice.ArchiveResult, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Replaced archived invoice %s; previous version kept at %s\n",
			invoice.DisplayPath(backup.Path, baseDir),
			invoice.DisplayPath(backup.BackupPath, baseDir),
		)
	}
}
