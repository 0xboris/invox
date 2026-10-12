// Package shared holds what the invoice commands share: required-input
// checks and the warnings about the archive.
package shared

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// RequireInput is the usage error when no invoice was given as the INVOICE
// argument or with -i, --input.
func RequireInput(invoicePath string) error {
	if strings.TrimSpace(invoicePath) == "" {
		return cmdutil.FlagErrorf("missing required input: INVOICE.yaml or -i, --input")
	}
	return nil
}

// RequireExtension is the usage error when the -o, --output path is set and
// does not end with ext.
func RequireExtension(outputPath, ext string) error {
	if strings.TrimSpace(outputPath) != "" && filepath.Ext(outputPath) != ext {
		return cmdutil.FlagErrorf("-o, --output must end with %s", ext)
	}
	return nil
}

// maxListedSkippedArchiveFiles caps the files named in the skipped archive
// warning.
const maxListedSkippedArchiveFiles = 5

// WarnSkippedArchiveFiles warns on stderr about archived invoices that
// numbering ignored because they do not match numbering.pattern.
func WarnSkippedArchiveFiles(ios *iostreams.IOStreams, customerID string, paths []string, baseDir string) {
	if len(paths) == 0 {
		return
	}
	listed := make([]string, 0, maxListedSkippedArchiveFiles)
	for _, path := range paths[:min(len(paths), maxListedSkippedArchiveFiles)] {
		listed = append(listed, cmdutil.DisplayPath(path, baseDir))
	}
	list := strings.Join(listed, ", ")
	if more := len(paths) - len(listed); more > 0 {
		list += fmt.Sprintf(" and %d more", more)
	}
	fmt.Fprintf(ios.ErrOut, "warning: numbering ignored %d archived invoice(s) for %s that do not match numbering.pattern: %s\n", len(paths), customerID, list)
	fmt.Fprintf(ios.ErrOut, "If they are obsolete, move them out of the archive or rename them to another extension. To continue their sequence, set numbering.start (or customers.%s.numbering.start) to the next number.\n", customerID)
}

// WarnUnread warns on stderr about the archived invoices invox no longer
// reads, one file per line, so the user can convert them. It prints nothing
// when there are none.
func WarnUnread(ios *iostreams.IOStreams, unread billing.Unread, baseDir string) {
	n := len(unread.Markdown)
	if n == 0 {
		return
	}
	dir := cmdutil.DisplayPath(unread.Dir, baseDir)
	if n == 1 {
		fmt.Fprintf(ios.ErrOut, "warning: 1 Markdown invoice in %s is no longer read; convert it to .yaml to include it:\n", dir)
	} else {
		fmt.Fprintf(ios.ErrOut, "warning: %d Markdown invoices in %s are no longer read; convert them to .yaml to include them:\n", n, dir)
	}
	for _, path := range unread.Markdown {
		fmt.Fprintf(ios.ErrOut, "  %s\n", cmdutil.DisplayPath(path, baseDir))
	}
}

// WarnArchivedDuplicate warns on stderr when err, the duplicate check of
// billing.ValidateResult, found the invoice's number on an archived invoice
// or could not check. It never fails validation.
func WarnArchivedDuplicate(ios *iostreams.IOStreams, err error, invoicePath, baseDir string) {
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
		cmdutil.DisplayPath(duplicate.ArchivedPath, baseDir),
		cmdutil.DisplayPath(invoicePath, baseDir),
	)
}

// TakeInput is the cobra.PositionalArgs of an invoice command. It makes the
// INVOICE argument, the only one the command takes, the invoice in
// invoicePath. It is a usage error when -i, --input names a different file.
// Paths relative to getwd count as the same file as their absolute form.
func TakeInput(getwd func() (string, error), invoicePath *string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if strings.TrimSpace(*invoicePath) == "" {
			*invoicePath = args[0]
		} else {
			cwd, err := getwd()
			if err != nil {
				return err
			}
			if cmdutil.AbsPath(cwd, args[0]) != cmdutil.AbsPath(cwd, *invoicePath) {
				return cmdutil.FlagErrorf("the INVOICE argument %s and -i, --input %s name different files; pass only one", args[0], *invoicePath)
			}
		}
		return cmdutil.MaximumArgs(1)(cmd, args)
	}
}
