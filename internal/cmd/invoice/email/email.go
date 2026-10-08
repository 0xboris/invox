// Package email is the `invox email` command, also run as `invox send`.
package email

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// EmailOptions is what email needs: its streams, the programs that show the
// draft, the user directories and the parsed flags.
type EmailOptions struct {
	IO     *iostreams.IOStreams
	Opener *opener.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer *applemail.Composer
	Host   func() invoice.Host
	Getwd  func() (string, error)
	Now    func() time.Time

	InvoicePath   string
	PDFPath       string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	To            string
	Subject       string
	Force         bool
	DryRun        bool
}

// NewCmdEmail returns the email command. runF replaces emailRun in tests.
func NewCmdEmail(f *cmdutil.Factory, runF func(context.Context, *EmailOptions) error) *cobra.Command {
	opts := &EmailOptions{IO: f.IOStreams, Opener: f.Opener, Mailer: f.Mailer, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
	cmd := &cobra.Command{
		Use:     "email [INVOICE.yaml | INVOICE.pdf]",
		Aliases: []string{"send"},
		Short:   "Create an email draft and open it in the default mail app",
		Long: `Create an email draft and open it in the default mail app.

Required inputs:
  INVOICE.yaml, INVOICE.pdf, or -i, --input PATH  Path to the invoice YAML or built PDF file

Default output:
  <input name>.eml in a new temporary directory, removed after 24 hours

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			`  invoice PDF: input path with .pdf extension by default, or the input itself when the input is a PDF

Behavior:
  Accepts either the invoice YAML file or the built PDF as input.
  When the input is a PDF, the matching YAML file is resolved from the same basename.
  The PDF lookup checks next to the PDF first, then archive.dir.
  Requires invoice.status to be built or archived and the PDF attachment to exist.
  On macOS, opens an editable compose window in Apple Mail with the PDF attached.
  If -o is set, or on non-macOS platforms, writes a .eml draft file and opens it.
  Without -o, the draft is written to a new temporary directory. A later run removes it after 24 hours.
  With -o, the draft is kept. An existing -o file is not replaced unless --force is set.
  Does not send the email and does not change invoice.status.
`,
		Example: `$ invox email invoice.yaml
$ invox email invoice.pdf
$ invox email invoice.yaml --to billing@example.com
$ invox email invoice.yaml --dry-run
$ invox email invoices/2026-0021.yaml -p out/2026-0021.pdf -o drafts/2026-0021.eml -c customers.yaml -u issuer.yaml
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if rest := shared.TakeInput(&opts.InvoicePath, args); len(rest) > 0 {
				return cmdutil.FlagErrorf("email", "unexpected arguments: %s", strings.Join(rest, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(opts); err != nil {
				return err
			}
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return emailRun(cmd.Context(), opts, cmd.Flags().Changed("output"))
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML or PDF file")
	cmd.Flags().StringVarP(&opts.PDFPath, "pdf", "p", "", "Path to the invoice PDF (default: the input with .pdf)")
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Write the draft to this .eml file instead of a temporary one (must end with .eml)")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.Flags().StringVarP(&opts.IssuerPath, "issuer", "u", "", "Path to issuer.yaml")
	cmd.Flags().StringVar(&opts.To, "to", "", "Recipient email override")
	cmd.Flags().StringVar(&opts.Subject, "subject", "", "Email subject override, supports placeholders")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite an existing output file")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the recipient, subject and attachment, and neither write nor open a draft")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml", "pdf")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml", "pdf")
	_ = cmd.MarkFlagFilename("pdf", "pdf")
	_ = cmd.MarkFlagFilename("output", "eml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	_ = cmd.RegisterFlagCompletionFunc("to", cobra.NoFileCompletions)
	_ = cmd.RegisterFlagCompletionFunc("subject", cobra.NoFileCompletions)
	return cmd
}

// validate checks the flags before any support file is read, as render does.
func validate(opts *EmailOptions) error {
	if strings.TrimSpace(opts.InvoicePath) == "" {
		return cmdutil.FlagErrorf("email", "missing required input: INVOICE.yaml, INVOICE.pdf, or -i, --input")
	}
	if err := shared.RequireExtension("email", opts.OutputPath, ".eml"); err != nil {
		return err
	}
	switch strings.ToLower(filepath.Ext(opts.InvoicePath)) {
	case ".yaml", ".yml", ".pdf":
	default:
		return cmdutil.FlagErrorf("email", "input must end with .yaml, .yml, or .pdf")
	}
	if strings.TrimSpace(opts.PDFPath) != "" && filepath.Ext(opts.PDFPath) != ".pdf" {
		return cmdutil.FlagErrorf("email", "-p, --pdf must end with .pdf")
	}
	return nil
}

const (
	draftDirPrefix = "invox-email-"
	// draftMaxAge is how long a temporary draft stays for the mail app
	// before a later run of invox email removes it.
	draftMaxAge = 24 * time.Hour
)

func emailRun(ctx context.Context, opts *EmailOptions, explicitOutput bool) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "email", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	issuerPath, err := cmdutil.SupportPath(h, "email", invoice.Issuer, opts.IssuerPath, baseDir)
	if err != nil {
		return err
	}
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)
	pdfPath := invoice.AbsPath(baseDir, orDefault(opts.PDFPath, shared.ReplaceExt(opts.InvoicePath, ".pdf")))
	outputPath := invoice.AbsPath(baseDir, orDefault(opts.OutputPath, shared.ReplaceExt(opts.InvoicePath, ".eml")))

	paths, err := h.ResolveEmailDraftPaths(invoicePath, pdfPath, outputPath)
	if err != nil {
		return err
	}
	message, err := h.PrepareInvoiceEmail(customersPath, issuerPath, paths.InvoicePath, paths.PDFPath, opts.To, opts.Subject)
	if err != nil {
		return err
	}
	outputExists := func(outputPath string, err error) error {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%s already exists; pass --force or choose another -o path", invoice.DisplayPath(outputPath, baseDir))
		}
		return err
	}
	draft := func(outputPath string, overwrite bool) error {
		_, err := h.CreateInvoiceEmailDraft(opts.Now(), customersPath, issuerPath, paths.InvoicePath, paths.PDFPath, outputPath, overwrite, opts.To, opts.Subject)
		return outputExists(outputPath, err)
	}

	if opts.DryRun {
		if explicitOutput {
			if err := outputExists(paths.OutputPath, invoice.CheckEmailDraftOutput(paths.OutputPath, opts.Force)); err != nil {
				return err
			}
		}
		fmt.Fprintf(opts.IO.ErrOut, "Would open email draft for %s (%s) to %s\n", message.CustomerID, message.InvoiceNumber, message.Recipient)
		fmt.Fprintf(opts.IO.ErrOut, "Subject: %s\nAttachment: %s\n", message.Subject, invoice.DisplayPath(message.AttachmentPath, baseDir))
		if explicitOutput {
			fmt.Fprintf(opts.IO.ErrOut, "Would write the draft to %s\n", invoice.DisplayPath(paths.OutputPath, baseDir))
			fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(paths.OutputPath, baseDir))
		}
		return nil
	}

	switch {
	case opts.Mailer != nil && !explicitOutput:
		if err := opts.Mailer.Compose(ctx, applemail.Message{
			To:         message.Recipient,
			Subject:    message.Subject,
			Body:       message.Body,
			Attachment: message.AttachmentPath,
			Sender:     message.SenderAddress,
		}); err != nil {
			return fmt.Errorf("failed to open editable email draft: %w", err)
		}
	case explicitOutput:
		if err := draft(paths.OutputPath, opts.Force); err != nil {
			return err
		}
		if err := opts.Opener.Open(ctx, paths.OutputPath); err != nil {
			return fmt.Errorf("created %s but failed to open it: %w", invoice.DisplayPath(paths.OutputPath, baseDir), err)
		}
	default:
		pruneEmailDrafts(os.TempDir(), opts.Now().Add(-draftMaxAge))
		draftDir, err := os.MkdirTemp("", draftDirPrefix+"*")
		if err != nil {
			return fmt.Errorf("create temporary draft directory: %w", err)
		}
		draftPath := filepath.Join(draftDir, filepath.Base(paths.OutputPath))
		if err := draft(draftPath, false); err != nil {
			_ = os.RemoveAll(draftDir)
			return err
		}
		// The draft stays in its temporary directory: the mail app can read it
		// after the opener returns, so invox cannot know when to delete it.
		// pruneEmailDrafts removes it on a run a day later.
		if err := opts.Opener.Open(ctx, draftPath); err != nil {
			_ = os.RemoveAll(draftDir)
			return fmt.Errorf("failed to open email draft: %w", err)
		}
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened email draft for %s (%s) to %s\n", message.CustomerID, message.InvoiceNumber, message.Recipient)
	if explicitOutput {
		fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(paths.OutputPath, baseDir))
	}
	return nil
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// pruneEmailDrafts removes the temporary draft directories in dir that earlier
// runs left behind and that were last modified before cutoff. It skips
// anything that is not a directory, so a symlink is never followed.
func pruneEmailDrafts(dir string, cutoff time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), draftDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, entry.Name()))
	}
}
