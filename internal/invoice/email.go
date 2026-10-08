package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/email"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/money"
)

type EmailDraftResult struct {
	OutputPath    string
	Recipient     string
	Subject       string
	CustomerID    string
	InvoiceNumber string
}

// EmailMessage is the draft of an invoice email and the invoice it is for.
type EmailMessage struct {
	email.Draft
	CustomerID    string
	InvoiceNumber string
}

type EmailDraftPaths struct {
	InvoicePath string
	PDFPath     string
	OutputPath  string
}

func (h Host) ResolveEmailDraftPaths(inputPath, pdfPath, outputPath string) (EmailDraftPaths, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return EmailDraftPaths{}, fmt.Errorf("input path is required")
	}

	paths := EmailDraftPaths{
		PDFPath:    strings.TrimSpace(pdfPath),
		OutputPath: strings.TrimSpace(outputPath),
	}

	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".pdf":
		resolvedInvoicePath, err := h.resolveInvoicePathForPDF(inputPath)
		if err != nil {
			return EmailDraftPaths{}, err
		}
		paths.InvoicePath = resolvedInvoicePath
		if paths.PDFPath == "" {
			paths.PDFPath = inputPath
		}
	case ".yaml", ".yml":
		paths.InvoicePath = inputPath
		if paths.PDFPath == "" {
			paths.PDFPath = replaceFileExtension(inputPath, ".pdf")
		}
	default:
		return EmailDraftPaths{}, fmt.Errorf("%s: input must end with .yaml, .yml, or .pdf", inputPath)
	}

	if paths.OutputPath == "" {
		paths.OutputPath = replaceFileExtension(inputPath, ".eml")
	}

	return paths, nil
}

func (h Host) resolveInvoicePathForPDF(pdfPath string) (string, error) {
	siblingCandidates := invoiceYAMLCandidatesForPDF(pdfPath)
	if path := firstExistingPath(siblingCandidates...); path != "" {
		return path, nil
	}

	archivePath, err := h.archivedInvoicePathForPDF(pdfPath)
	if err != nil {
		return "", err
	}
	if archivePath != "" {
		return archivePath, nil
	}

	return "", fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", pdfPath)
}

func invoiceYAMLCandidatesForPDF(pdfPath string) []string {
	basePath := strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath))
	return []string{basePath + ".yaml", basePath + ".yml"}
}

func (h Host) archivedInvoicePathForPDF(pdfPath string) (string, error) {
	store, err := h.archiveStore()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return "", nil
	}

	basenames := make([]string, 0, 2)
	for _, path := range invoiceYAMLCandidatesForPDF(pdfPath) {
		basenames = append(basenames, filepath.Base(path))
	}
	if path := firstExistingPath(
		filepath.Join(store.Dir, basenames[0]),
		filepath.Join(store.Dir, basenames[1]),
	); path != "" {
		return path, nil
	}

	matches := make([]string, 0, 1)
	err = store.Walk(func(path string) error {
		name := filepath.Base(path)
		for _, basename := range basenames {
			if name == basename {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	sort.Strings(matches)
	return "", fmt.Errorf(
		"%s: multiple archived invoice YAML files match %s; pass the YAML path explicitly: %s",
		store.Dir,
		filepath.Base(replaceFileExtension(pdfPath, ".yaml")),
		strings.Join(matches, ", "),
	)
}

// CreateInvoiceEmailDraft writes the draft to outputPath. Unless overwrite is set, an
// existing outputPath is left untouched and the returned error matches fs.ErrExist.
func (h Host) CreateInvoiceEmailDraft(now time.Time, customersPath, issuerPath, invoicePath, pdfPath, outputPath string, overwrite bool, recipientOverride, subjectOverride string) (EmailDraftResult, error) {
	if err := CheckEmailDraftOutput(outputPath, overwrite); err != nil {
		return EmailDraftResult{}, err
	}

	emailMessage, err := h.PrepareInvoiceEmail(customersPath, issuerPath, invoicePath, pdfPath, recipientOverride, subjectOverride)
	if err != nil {
		return EmailDraftResult{}, err
	}

	pdfBytes, err := os.ReadFile(emailMessage.AttachmentPath)
	if err != nil {
		return EmailDraftResult{}, fmt.Errorf("read %s: %w", emailMessage.AttachmentPath, err)
	}

	message, err := email.Build(emailMessage.Draft, pdfBytes, now, fmt.Sprintf("invox-boundary-%d", now.UnixNano()))
	if err != nil {
		return EmailDraftResult{}, err
	}
	write := fsutil.WriteNewFile
	if overwrite {
		write = fsutil.WriteFile
	}
	if err := write(outputPath, message, fsutil.Public); err != nil {
		return EmailDraftResult{}, err
	}

	return EmailDraftResult{
		OutputPath:    outputPath,
		Recipient:     emailMessage.Recipient,
		Subject:       emailMessage.Subject,
		CustomerID:    emailMessage.CustomerID,
		InvoiceNumber: emailMessage.InvoiceNumber,
	}, nil
}

// CheckEmailDraftOutput returns an error matching fs.ErrExist when a draft
// would replace an existing outputPath and overwrite is not set.
func CheckEmailDraftOutput(outputPath string, overwrite bool) error {
	if overwrite {
		return nil
	}
	if _, err := os.Lstat(outputPath); err == nil {
		return &fs.PathError{Op: "write", Path: outputPath, Err: fs.ErrExist}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (h Host) PrepareInvoiceEmail(customersPath, issuerPath, invoicePath, pdfPath, recipientOverride, subjectOverride string) (EmailMessage, error) {
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		return EmailMessage{}, err
	}

	status := ctx.Invoice.Status.Trim()
	if status != "built" && status != "archived" {
		if status == "" {
			return EmailMessage{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft", invoicePath)
		}
		return EmailMessage{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft, got `%s`", invoicePath, status)
	}
	if _, err := os.Stat(pdfPath); err != nil {
		return EmailMessage{}, fmt.Errorf("read %s: %w", pdfPath, err)
	}

	recipient := strings.TrimSpace(recipientOverride)
	if recipient == "" {
		recipient = strings.TrimSpace(ctx.CustomerEmail)
	}
	if recipient == "" {
		return EmailMessage{}, fmt.Errorf("%s: recipient email is unavailable", invoicePath)
	}

	subject, err := h.invoiceEmailSubject(ctx, invoicePath, subjectOverride)
	if err != nil {
		return EmailMessage{}, err
	}
	body, err := h.invoiceEmailBodyText(ctx)
	if err != nil {
		return EmailMessage{}, err
	}

	return EmailMessage{
		Draft: email.Draft{
			Recipient:      recipient,
			Subject:        subject,
			Body:           body,
			SenderName:     ctx.Company.LegalCompanyName.Trim(),
			SenderAddress:  ctx.Company.Email.Trim(),
			AttachmentPath: pdfPath,
		},
		CustomerID:    ctx.CustomerID,
		InvoiceNumber: ctx.InvoiceNumber,
	}, nil
}

func (h Host) invoiceEmailSubject(ctx *Context, invoicePath, subjectOverride string) (string, error) {
	template := strings.TrimSpace(subjectOverride)
	if template == "" {
		cfg, err := h.Config()
		if err != nil {
			return "", err
		}
		template = string(cfg.Email.Subject)
	}
	subject, err := email.Subject(template, emailFields(ctx))
	if err != nil {
		return "", fmt.Errorf("%s: %w", invoicePath, err)
	}
	return subject, nil
}

func (h Host) invoiceEmailBodyText(ctx *Context) (string, error) {
	cfg, err := h.Config()
	if err != nil {
		return "", err
	}
	return email.Body(string(cfg.Email.Body), emailFields(ctx)), nil
}

// emailFields is what the email placeholders stand for in ctx.
func emailFields(ctx *Context) email.Fields {
	return email.Fields{
		CustomerName:      ctx.Customer.DisplayName(),
		Greeting:          ctx.Customer.emailGreeting(),
		ContactPerson:     ctx.Customer.contactPerson(),
		CustomerID:        ctx.CustomerID,
		InvoiceNumber:     ctx.InvoiceNumber,
		IssueDate:         ctx.Invoice.IssueDate.String(),
		DueDate:           ctx.Invoice.DueDate.String(),
		TotalAmount:       emailMoney(ctx.TotalCents, ctx.Currency),
		OutstandingAmount: emailMoney(ctx.OutstandingCents, ctx.Currency),
		PaymentTermsText:  ctx.Payment.PaymentTermsText.Trim(),
		IssuerName:        ctx.Company.LegalCompanyName.Trim(),
	}
}

func emailMoney(cents int64, currency string) string {
	return money.FormatCents(cents) + " " + currency
}
