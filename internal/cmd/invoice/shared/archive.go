package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// ConfirmReplace returns the question archiving asks before it replaces
// archived files, for billing.ArchiveOptions.Confirm. errorPrefix starts the
// messages it reports. It returns a *cmdutil.FlagError when there is no
// terminal to ask on, and cmdutil.CancelError when the user declines or ctx
// is cancelled at the prompt.
func ConfirmReplace(ctx context.Context, ios *iostreams.IOStreams, command, invoicePath, baseDir, errorPrefix string) func(*billing.ArchiveReplaceError) error {
	return func(replaceErr *billing.ArchiveReplaceError) error {
		paths := make([]string, 0, len(replaceErr.Paths))
		for _, path := range replaceErr.Paths {
			paths = append(paths, cmdutil.DisplayPath(path, baseDir))
		}
		replaced := strings.Join(paths, ", ")
		if !ios.CanPrompt() {
			return cmdutil.FlagErrorf(
				command,
				"%sarchiving %s replaces archived invoice %s; pass --yes to replace it (%s)",
				errorPrefix,
				cmdutil.DisplayPath(invoicePath, baseDir),
				replaced,
				cmdutil.WhyNoPrompt(ios),
			)
		}
		question := fmt.Sprintf(
			"Replace archived invoice %s? The previous version is kept in %s.",
			replaced,
			cmdutil.DisplayPath(replaceErr.HistoryDir, baseDir),
		)
		confirmed, err := cmdutil.Confirm(ctx, ios, question)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(ios.ErrOut, "%snot archived; the archive was not changed; pass --yes to replace without asking\n", errorPrefix)
			return cmdutil.CancelError
		}
		return nil
	}
}

// PrintArchiveReplacements tells the user on stderr which archived files were
// replaced and where their previous versions are.
func PrintArchiveReplacements(ios *iostreams.IOStreams, result billing.ArchiveResult, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Replaced archived invoice %s; previous version kept at %s\n",
			cmdutil.DisplayPath(backup.Path, baseDir),
			cmdutil.DisplayPath(backup.BackupPath, baseDir),
		)
	}
}

// PrintArchivePreview tells the user on stderr what archiving invoicePath
// would do, from the result of a dry run.
func PrintArchivePreview(ios *iostreams.IOStreams, result billing.ArchiveResult, invoicePath, baseDir string) {
	for _, backup := range result.Replaced {
		fmt.Fprintf(
			ios.ErrOut,
			"Would replace archived invoice %s; the previous version would be kept in %s\n",
			cmdutil.DisplayPath(backup.Path, baseDir),
			cmdutil.DisplayPath(result.HistoryDir, baseDir),
		)
	}
	fmt.Fprintf(ios.ErrOut, "Would archive %s -> %s\n", cmdutil.DisplayPath(invoicePath, baseDir), cmdutil.DisplayPath(result.Path, baseDir))
}
