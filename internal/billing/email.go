package billing

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/money"
)

// EmailRequest says which invoice DraftEmail drafts an email for.
type EmailRequest struct {
	// Invoice is the invoice file. It is "" when the user named the built
	// PDF instead: FromPDF is then that PDF, and the invoice is the YAML
	// file with its name next to it, else in the archive.
	Invoice string
	FromPDF string
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
	// Unread is what the archive walk for the PDF's invoice could not read.
	Unread Unread
}

// DraftEmail drafts an email with the invoice's PDF attached, for an
// invoice that is built or archived. It never sends it.
func (s *Service) DraftEmail(ctx context.Context, req EmailRequest) (EmailResult, error) {
	customersPath, issuerPath, err := s.locateParties()
	if err != nil {
		return EmailResult{}, err
	}
	invoicePath, unread, err := s.emailInvoice(req)
	if err != nil {
		return EmailResult{}, err
	}
	pdfPath, outputPath := req.PDF, req.Output
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
	if err := s.Mailer.CheckAttachment(pdfPath); err != nil {
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
		Unread: unread,
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

// emailInvoice returns the invoice of req: the one it names, else the one
// its PDF was built from, with what the archive walk could not read.
func (s *Service) emailInvoice(req EmailRequest) (string, Unread, error) {
	if req.Invoice != "" {
		return req.Invoice, Unread{}, nil
	}
	invoicePath, unread, err := s.Archives.Source(req.FromPDF)
	if err != nil {
		return "", unread, err
	}
	if invoicePath == "" {
		return "", unread, fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", req.FromPDF)
	}
	return invoicePath, unread, nil
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
