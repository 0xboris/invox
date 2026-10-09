// Package email is the `invox email` command.
package email

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
)

// EmailOptions is what email needs: its streams, the use cases and the
// parsed flags. The mailer the use cases get opens the draft.
type EmailOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

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
	opts := &EmailOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:        "email [INVOICE.yaml | INVOICE.pdf]",
		SuggestFor: []string{"send"},
		Short:      "Create an email draft and open it in the default mail app",
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
		Args: shared.TakeInput(opts.Getwd, &opts.InvoicePath),
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
		return cmdutil.FlagErrorf("missing required input: INVOICE.yaml, INVOICE.pdf, or -i, --input")
	}
	if err := shared.RequireExtension(opts.OutputPath, ".eml"); err != nil {
		return err
	}
	switch strings.ToLower(filepath.Ext(opts.InvoicePath)) {
	case ".yaml", ".yml", ".pdf":
	default:
		return cmdutil.FlagErrorf("input must end with .yaml, .yml, or .pdf")
	}
	if strings.TrimSpace(opts.PDFPath) != "" && filepath.Ext(opts.PDFPath) != ".pdf" {
		return cmdutil.FlagErrorf("-p, --pdf must end with .pdf")
	}
	return nil
}

func emailRun(ctx context.Context, opts *EmailOptions, explicitOutput bool) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(cmdutil.Files{
		Customers:   cmdutil.AbsFlag(baseDir, opts.CustomersPath),
		Issuer:      cmdutil.AbsFlag(baseDir, opts.IssuerPath),
		EmailOutput: explicitOutput,
	})
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)
	pdfPath := cmdutil.AbsPath(baseDir, orDefault(opts.PDFPath, cmdutil.ReplaceExt(opts.InvoicePath, ".pdf")))
	outputPath := cmdutil.AbsPath(baseDir, orDefault(opts.OutputPath, cmdutil.ReplaceExt(opts.InvoicePath, ".eml")))

	request := billing.EmailRequest{
		Invoice:   invoicePath,
		PDF:       pdfPath,
		Output:    outputPath,
		Keep:      explicitOutput,
		To:        opts.To,
		Subject:   opts.Subject,
		Overwrite: opts.Force,
		DryRun:    opts.DryRun,
	}
	if strings.EqualFold(filepath.Ext(invoicePath), ".pdf") {
		request.Invoice, request.FromPDF = "", invoicePath
	}
	result, err := svc.DraftEmail(ctx, request)
	if err != nil {
		return outputExists(cmdutil.UsageError(err), baseDir)
	}
	shared.WarnUnread(opts.IO, result.Unread, baseDir)
	message := result.Message

	if opts.DryRun {
		fmt.Fprintf(opts.IO.ErrOut, "Would open email draft for %s (%s) to %s\n", result.CustomerID, result.Number, message.To)
		fmt.Fprintf(opts.IO.ErrOut, "Subject: %s\nAttachment: %s\n", message.Subject, cmdutil.DisplayPath(message.Attachment, baseDir))
		if explicitOutput {
			fmt.Fprintf(opts.IO.ErrOut, "Would write the draft to %s\n", cmdutil.DisplayPath(message.Output, baseDir))
			fmt.Fprintln(opts.IO.Out, cmdutil.DisplayPath(message.Output, baseDir))
		}
		return nil
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened email draft for %s (%s) to %s\n", result.CustomerID, result.Number, message.To)
	if explicitOutput {
		fmt.Fprintln(opts.IO.Out, cmdutil.DisplayPath(message.Output, baseDir))
	}
	return nil
}

// outputExists words an error about the draft's file for the email
// command.
func outputExists(err error, baseDir string) error {
	var isDir *billing.OutputIsDirError
	if errors.As(err, &isDir) {
		return fmt.Errorf("%s is a directory; choose another -o path", cmdutil.DisplayPath(isDir.Path, baseDir))
	}
	var pathErr *fs.PathError
	if errors.Is(err, fs.ErrExist) && errors.As(err, &pathErr) {
		return fmt.Errorf("%s already exists; pass --force or choose another -o path", cmdutil.DisplayPath(pathErr.Path, baseDir))
	}
	return err
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
