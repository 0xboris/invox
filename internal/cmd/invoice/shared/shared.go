// Package shared holds what the invoice commands share: required-input
// checks and the warnings about the archive.
package shared

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/store"
)

// RequireInput is the usage error of command when no invoice was given as
// the INVOICE argument or with -i, --input.
func RequireInput(command, invoicePath string) error {
	if strings.TrimSpace(invoicePath) == "" {
		return cmdutil.FlagErrorf(command, "missing required input: INVOICE.yaml or -i, --input")
	}
	return nil
}

// RequireExtension is the usage error of command when the -o, --output path
// is set and does not end with ext.
func RequireExtension(command, outputPath, ext string) error {
	if strings.TrimSpace(outputPath) != "" && filepath.Ext(outputPath) != ext {
		return cmdutil.FlagErrorf(command, "-o, --output must end with %s", ext)
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
		listed = append(listed, store.DisplayPath(path, baseDir))
	}
	list := strings.Join(listed, ", ")
	if more := len(paths) - len(listed); more > 0 {
		list += fmt.Sprintf(" and %d more", more)
	}
	fmt.Fprintf(ios.ErrOut, "warning: numbering ignored %d archived invoice(s) for %s that do not match numbering.pattern: %s\n", len(paths), customerID, list)
	fmt.Fprintf(ios.ErrOut, "If they are obsolete, move them out of the archive or rename them to another extension. To continue their sequence, set numbering.start (or customers.%s.numbering.start) to the next number.\n", customerID)
}

// WarnArchivedDuplicate warns on stderr when the invoice's number is already
// used by an archived invoice. It never fails validation.
func WarnArchivedDuplicate(ios *iostreams.IOStreams, h store.Host, invoicePath, baseDir string) {
	err := h.CheckArchivedNumberUnique(invoicePath)
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
		store.DisplayPath(duplicate.ArchivedPath, baseDir),
		store.DisplayPath(invoicePath, baseDir),
	)
}

// TakeInput makes the INVOICE argument, the first of args, the invoice of
// command. It is a usage error when -i, --input names a different file, or
// when more arguments follow. Paths relative to getwd count as the same
// file as their absolute form.
func TakeInput(command string, getwd func() (string, error), invoicePath *string, args []string) error {
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
		if store.AbsPath(cwd, args[0]) != store.AbsPath(cwd, *invoicePath) {
			return cmdutil.FlagErrorf(command, "the INVOICE argument %s and -i, --input %s name different files; pass only one", args[0], *invoicePath)
		}
	}
	if rest := args[1:]; len(rest) > 0 {
		return cmdutil.FlagErrorf(command, "unexpected arguments: %s", strings.Join(rest, " "))
	}
	return nil
}
