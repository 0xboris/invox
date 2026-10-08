package billing

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/money"
)

// EmailRequest says which invoice DraftEmail drafts an email for.
type EmailRequest struct {
	// Input is the invoice, or its built PDF, whose invoice is found next
	// to it or in the archive.
	Input string
	// PDF is the attachment, Output the .eml draft.
	PDF    string
	Output string
	// Keep writes the draft to Output and keeps it; otherwise it goes to
	// a temporary file, or to the mail app.
	Keep bool
	// To and Subject replace the recipient and the subject template.
	To      string
	Subject string
	// Overwrite replaces an existing Output.
	Overwrite bool
	// DryRun runs every check and drafts nothing.
	DryRun bool
}

// EmailResult is the drafted email.
type EmailResult struct {
	CustomerID string
	Number     string
	Message    Message
	// Draft is where the draft went; unset in a dry run.
	Draft Draft
}

// DraftEmail drafts an email with the invoice's PDF attached, for an
// invoice that is built or archived. It never sends it.
func (s *Service) DraftEmail(ctx context.Context, req EmailRequest) (EmailResult, error) {
	customersPath, issuerPath, err := s.locateParties()
	if err != nil {
		return EmailResult{}, err
	}
	invoicePath, pdfPath, outputPath, err := s.EmailPaths(req.Input, req.PDF, req.Output)
	if err != nil {
		return EmailResult{}, err
	}
	inv, err := s.loadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		return EmailResult{}, err
	}
	status := invoice.Status(inv.Header.Status.Trim())
	if !status.Allows(invoice.Emailing) {
		if status == "" {
			return EmailResult{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft", invoicePath)
		}
		return EmailResult{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft, got `%s`", invoicePath, status)
	}
	if err := s.Invoices.Stat(pdfPath); err != nil {
		return EmailResult{}, fmt.Errorf("read %s: %w", pdfPath, err)
	}

	recipient := strings.TrimSpace(req.To)
	if recipient == "" {
		recipient = strings.TrimSpace(inv.CustomerEmail)
	}
	if recipient == "" {
		return EmailResult{}, fmt.Errorf("%s: recipient email is unavailable", invoicePath)
	}
	settings, err := s.Settings()
	if err != nil {
		return EmailResult{}, err
	}
	subjectTemplate := strings.TrimSpace(req.Subject)
	if subjectTemplate == "" {
		subjectTemplate = settings.EmailSubject
	}
	fields := emailFields(inv)
	subjectText, err := subject(subjectTemplate, fields)
	if err != nil {
		return EmailResult{}, fmt.Errorf("%s: %w", invoicePath, err)
	}

	result := EmailResult{
		CustomerID: inv.CustomerID,
		Number:     inv.InvoiceNumber,
		Message: Message{
			To:          recipient,
			Subject:     subjectText,
			Body:        body(settings.EmailBody, fields),
			FromName:    inv.Company.LegalCompanyName.Trim(),
			FromAddress: inv.Company.Email.Trim(),
			Attachment:  pdfPath,
			Output:      outputPath,
			Temporary:   !req.Keep,
			Overwrite:   req.Keep && req.Overwrite,
			Date:        s.Now(),
		},
	}
	if req.DryRun {
		if req.Keep {
			if err := s.Mailer.Check(result.Message); err != nil {
				return EmailResult{}, err
			}
		}
		return result, nil
	}
	result.Draft, err = s.Mailer.Draft(ctx, result.Message)
	if err != nil {
		return EmailResult{}, err
	}
	return result, nil
}

// EmailPaths returns the invoice, PDF and draft of an email for input, an
// invoice YAML file or its PDF.
func (s *Service) EmailPaths(input, pdf, output string) (string, string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", "", fmt.Errorf("input path is required")
	}
	pdf, output = strings.TrimSpace(pdf), strings.TrimSpace(output)

	var invoicePath string
	switch strings.ToLower(filepath.Ext(input)) {
	case ".pdf":
		resolved, err := s.invoiceForPDF(input)
		if err != nil {
			return "", "", "", err
		}
		invoicePath = resolved
		if pdf == "" {
			pdf = input
		}
	case ".yaml", ".yml":
		invoicePath = input
		if pdf == "" {
			pdf = replaceExt(input, ".pdf")
		}
	default:
		return "", "", "", fmt.Errorf("%s: input must end with .yaml, .yml, or .pdf", input)
	}
	if output == "" {
		output = replaceExt(input, ".eml")
	}
	return invoicePath, pdf, output, nil
}

// invoiceForPDF finds the invoice of the PDF at pdfPath: next to it, else in
// the archive.
func (s *Service) invoiceForPDF(pdfPath string) (string, error) {
	base := strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath))
	candidates := []string{base + ".yaml", base + ".yml"}
	for _, candidate := range candidates {
		if s.Invoices.Exists(candidate) {
			return candidate, nil
		}
	}
	archived, err := s.Archives.FindFile(filepath.Base(candidates[0]), filepath.Base(candidates[1]))
	if err != nil {
		return "", err
	}
	if archived != "" {
		return archived, nil
	}
	return "", fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", pdfPath)
}

// emailFields is what the email placeholders stand for in ctx.
func emailFields(ctx *invoice.Context) EmailFields {
	return EmailFields{
		CustomerName:      ctx.Customer.DisplayName(),
		Greeting:          ctx.Customer.Greeting(),
		ContactPerson:     ctx.Customer.Contact(),
		CustomerID:        ctx.CustomerID,
		InvoiceNumber:     ctx.InvoiceNumber,
		IssueDate:         ctx.Header.IssueDate.String(),
		DueDate:           ctx.Header.DueDate.String(),
		TotalAmount:       emailMoney(ctx.TotalCents, ctx.Currency),
		OutstandingAmount: emailMoney(ctx.OutstandingCents, ctx.Currency),
		PaymentTermsText:  ctx.Payment.PaymentTermsText.Trim(),
		IssuerName:        ctx.Company.LegalCompanyName.Trim(),
	}
}

func emailMoney(cents int64, currency string) string {
	return money.FormatCents(cents) + " " + currency
}

// replaceExt returns path with its extension replaced by ext.
func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}
