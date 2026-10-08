package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/tableprinter"
)

func runEmail(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
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

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	paths, err := h.ResolveEmailDraftPaths(opts.InvoicePath, opts.PDFPath, opts.OutputPath)
	if err != nil {
		return err
	}

	emailMessage, err := h.PrepareInvoiceEmail(
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

	if f.Mailer != nil && !explicitOutputPath {
		if err := f.Mailer.Compose(ctx, applemail.Message{
			To:         emailMessage.Recipient,
			Subject:    emailMessage.Subject,
			Body:       emailMessage.Body,
			Attachment: emailMessage.AttachmentPath,
			Sender:     emailMessage.SenderAddress,
		}); err != nil {
			return fmt.Errorf("failed to open editable email draft: %w", err)
		}
	} else if explicitOutputPath {
		if err := writeEmailDraft(h, e.Now(), opts, paths, paths.OutputPath, opts.OverwriteOutput); err != nil {
			return err
		}
		if err := f.Opener.Open(ctx, paths.OutputPath); err != nil {
			return fmt.Errorf("created %s but failed to open it: %w", invoice.DisplayPath(paths.OutputPath, opts.BaseDir), err)
		}
	} else {
		pruneEmailDrafts(os.TempDir(), e.Now().Add(-emailDraftMaxAge))
		draftDir, err := os.MkdirTemp("", emailDraftDirPrefix+"*")
		if err != nil {
			return fmt.Errorf("create temporary draft directory: %w", err)
		}
		draftPath := filepath.Join(draftDir, filepath.Base(paths.OutputPath))
		if err := writeEmailDraft(h, e.Now(), opts, paths, draftPath, false); err != nil {
			_ = os.RemoveAll(draftDir)
			return err
		}
		// The draft stays in its temporary directory: the mail app can read it
		// after the opener returns, so invox cannot know when to delete it.
		// pruneEmailDrafts removes it on a run a day later.
		if err := f.Opener.Open(ctx, draftPath); err != nil {
			_ = os.RemoveAll(draftDir)
			return fmt.Errorf("failed to open email draft: %w", err)
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

const (
	emailDraftDirPrefix = "invox-email-"
	// emailDraftMaxAge is how long a temporary draft stays for the mail app
	// before a later run of invox email removes it.
	emailDraftMaxAge = 24 * time.Hour
)

// pruneEmailDrafts removes the temporary draft directories in dir that earlier
// runs left behind and that were last modified before cutoff. It skips
// anything that is not a directory, so a symlink is never followed.
func pruneEmailDrafts(dir string, cutoff time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), emailDraftDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, entry.Name()))
	}
}

// writeEmailDraft writes the .eml draft to outputPath.
func writeEmailDraft(h invoice.Host, now time.Time, opts invoice.Options, paths invoice.EmailDraftPaths, outputPath string, overwrite bool) error {
	_, err := h.CreateInvoiceEmailDraft(
		now,
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

func runBuild(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
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

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}

	inv, err := invoice.LoadContext(opts.CustomersPath, opts.IssuerPath, opts.InvoicePath)
	if err != nil {
		return err
	}

	if err := h.BuildInvoicePDF(ctx, f.Compiler.Build, opts.TemplatePath, opts.OutputPath, inv); err != nil {
		var execErr *run.ExecError
		if errors.As(err, &execErr) {
			return &cmdutil.ExecError{Program: "tectonic", Code: execErr.Code, Err: err}
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
		result, err := archiveWithConfirmation(ctx, ios, h, e.Now, spec, opts, errorPrefix)
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
			inv.CustomerID,
			inv.InvoiceNumber,
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
		inv.CustomerID,
		inv.InvoiceNumber,
	)
	fmt.Fprintln(ios.Out, invoice.DisplayPath(opts.OutputPath, opts.BaseDir))
	return nil
}

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

	result, err := archiveWithConfirmation(ctx, ios, h, e.Now, spec, opts, "")
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
