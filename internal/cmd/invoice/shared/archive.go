package shared

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/store"
)

// ArchiveWithConfirmation archives the invoice at invoicePath and, when that replaces
// archived files, asks first unless assumeYes (--yes) is set. errorPrefix starts
// the messages it reports. It returns a *cmdutil.FlagError when there is no
// terminal to ask on, and cmdutil.CancelError when the user declines or ctx is
// cancelled at the prompt.
func ArchiveWithConfirmation(ctx context.Context, ios *iostreams.IOStreams, h store.Host, now func() time.Time, command, invoicePath, baseDir string, assumeYes bool, errorPrefix string) (store.ArchiveResult, error) {
	archiveOpts := store.ArchiveOptions{Replace: assumeYes}
	result, err := h.ArchiveInvoice(now(), invoicePath, archiveOpts)
	var replaceErr *store.ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		return result, err
	}

	paths := make([]string, 0, len(replaceErr.Paths))
	for _, path := range replaceErr.Paths {
		paths = append(paths, store.DisplayPath(path, baseDir))
	}
	replaced := strings.Join(paths, ", ")
	if !ios.CanPrompt() {
		return store.ArchiveResult{}, cmdutil.FlagErrorf(
			command,
			"%sarchiving %s replaces archived invoice %s; pass --yes to replace it (%s)",
			errorPrefix,
			store.DisplayPath(invoicePath, baseDir),
			replaced,
			cmdutil.WhyNoPrompt(ios),
		)
	}
	question := fmt.Sprintf(
		"Replace archived invoice %s? The previous version is kept in %s.",
		replaced,
		store.DisplayPath(replaceErr.HistoryDir, baseDir),
	)
	confirmed, err := cmdutil.Confirm(ctx, ios, question)
	if err != nil {
		return store.ArchiveResult{}, err
	}
	if !confirmed {
		fmt.Fprintf(ios.ErrOut, "%snot archived; the archive was not changed; pass --yes to replace without asking\n", errorPrefix)
		return store.ArchiveResult{}, cmdutil.CancelError
	}

	archiveOpts.Replace = true
	return h.ArchiveInvoice(now(), invoicePath, archiveOpts)
}

// PrintArchiveReplacements tells the user on stderr which archived files were
// replaced and where their previous versions are.
func PrintArchiveReplacements(ios *iostreams.IOStreams, result store.ArchiveResult, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Replaced archived invoice %s; previous version kept at %s\n",
			store.DisplayPath(backup.Path, baseDir),
			store.DisplayPath(backup.BackupPath, baseDir),
		)
	}
}

// PrintArchivePreview tells the user on stderr what archiving invoicePath
// would do, from the result of a dry run.
func PrintArchivePreview(ios *iostreams.IOStreams, result store.ArchiveResult, invoicePath, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Would replace archived invoice %s; the previous version would be kept in %s\n",
			store.DisplayPath(backup.Path, baseDir),
			store.DisplayPath(result.HistoryDir, baseDir),
		)
	}
	fmt.Fprintf(ios.ErrOut, "Would archive %s -> %s\n", store.DisplayPath(invoicePath, baseDir), store.DisplayPath(result.Path, baseDir))
}
