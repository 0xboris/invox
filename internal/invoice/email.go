package invoice

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type EmailDraftResult struct {
	OutputPath    string
	Recipient     string
	Subject       string
	CustomerID    string
	InvoiceNumber string
}

type EmailMessage struct {
	Recipient      string
	Subject        string
	Body           string
	SenderName     string
	SenderAddress  string
	AttachmentPath string
	CustomerID     string
	InvoiceNumber  string
}

const (
	defaultEmailSubjectTemplate = "Invoice {invoice_number}"
	defaultEmailBodyTemplate    = `{email_greeting}

Please find attached invoice {invoice_number}.
Issue date: {issue_date}
Due date: {due_date}
Outstanding amount: {outstanding_amount}

Regards,
{issuer_name}
`
)

type EmailDraftPaths struct {
	InvoicePath string
	PDFPath     string
	OutputPath  string
}

func ResolveEmailDraftPaths(inputPath, pdfPath, outputPath string) (EmailDraftPaths, error) {
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
		resolvedInvoicePath, err := resolveInvoicePathForPDF(inputPath)
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

func resolveInvoicePathForPDF(pdfPath string) (string, error) {
	siblingCandidates := invoiceYAMLCandidatesForPDF(pdfPath)
	if path := firstExistingPath(siblingCandidates...); path != "" {
		return path, nil
	}

	archivePath, err := archivedInvoicePathForPDF(pdfPath)
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

func archivedInvoicePathForPDF(pdfPath string) (string, error) {
	archiveDir, err := ResolveArchiveDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return "", nil
	}

	info, err := os.Stat(archiveDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: archive.dir must point to a directory", archiveDir)
	}

	basenames := make([]string, 0, 2)
	for _, path := range invoiceYAMLCandidatesForPDF(pdfPath) {
		basenames = append(basenames, filepath.Base(path))
	}
	if path := firstExistingPath(
		filepath.Join(archiveDir, basenames[0]),
		filepath.Join(archiveDir, basenames[1]),
	); path != "" {
		return path, nil
	}

	matches := make([]string, 0, 1)
	err = filepath.WalkDir(archiveDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
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
		archiveDir,
		filepath.Base(replaceFileExtension(pdfPath, ".yaml")),
		strings.Join(matches, ", "),
	)
}

func CreateInvoiceEmailDraft(customersPath, issuerPath, invoicePath, pdfPath, outputPath, recipientOverride, subjectOverride string) (EmailDraftResult, error) {
	emailMessage, err := PrepareInvoiceEmail(customersPath, issuerPath, invoicePath, pdfPath, recipientOverride, subjectOverride)
	if err != nil {
		return EmailDraftResult{}, err
	}

	pdfBytes, err := os.ReadFile(emailMessage.AttachmentPath)
	if err != nil {
		return EmailDraftResult{}, fmt.Errorf("read %s: %w", emailMessage.AttachmentPath, err)
	}

	message, err := buildInvoiceEmailDraft(emailMessage, pdfBytes)
	if err != nil {
		return EmailDraftResult{}, err
	}
	if err := writeFileAtomic(outputPath, message, 0o644); err != nil {
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

func PrepareInvoiceEmail(customersPath, issuerPath, invoicePath, pdfPath, recipientOverride, subjectOverride string) (EmailMessage, error) {
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		return EmailMessage{}, err
	}

	status := strings.TrimSpace(asString(ctx.Invoice["status"]))
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

	subject, err := invoiceEmailSubject(ctx, invoicePath, subjectOverride)
	if err != nil {
		return EmailMessage{}, err
	}
	body, err := invoiceEmailBodyText(ctx)
	if err != nil {
		return EmailMessage{}, err
	}

	return EmailMessage{
		Recipient:      recipient,
		Subject:        subject,
		Body:           body,
		SenderName:     strings.TrimSpace(asString(ctx.IssuerCompany["legal_company_name"])),
		SenderAddress:  strings.TrimSpace(asString(ctx.IssuerCompany["email"])),
		AttachmentPath: pdfPath,
		CustomerID:     ctx.CustomerID,
		InvoiceNumber:  ctx.InvoiceNumber,
	}, nil
}

func buildInvoiceEmailDraft(emailMessage EmailMessage, pdfBytes []byte) ([]byte, error) {
	fromAddress := mail.Address{
		Name:    emailMessage.SenderName,
		Address: emailMessage.SenderAddress,
	}
	toAddress := mail.Address{
		Address: emailMessage.Recipient,
	}

	var buffer bytes.Buffer
	boundary := fmt.Sprintf("invox-boundary-%d", time.Now().UnixNano())

	fmt.Fprintf(&buffer, "From: %s\r\n", fromAddress.String())
	fmt.Fprintf(&buffer, "To: %s\r\n", toAddress.String())
	fmt.Fprintf(&buffer, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", emailMessage.Subject))
	fmt.Fprintf(&buffer, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buffer, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buffer, "X-Unsent: 1\r\n")
	fmt.Fprintf(&buffer, "Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buffer, "\r\n")

	writer := multipart.NewWriter(&buffer)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, err
	}

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {`text/plain; charset="utf-8"`},
		"Content-Transfer-Encoding": {"7bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(strings.ReplaceAll(emailMessage.Body, "\n", "\r\n"))); err != nil {
		return nil, err
	}

	filename := filepath.Base(emailMessage.AttachmentPath)
	attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {fmt.Sprintf(`application/pdf; name="%s"`, filename)},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, filename)},
	})
	if err != nil {
		return nil, err
	}
	if err := writeBase64MIME(attachmentPart, pdfBytes); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func invoiceEmailSubject(ctx *Context, invoicePath, subjectOverride string) (string, error) {
	template := strings.TrimSpace(subjectOverride)
	if template == "" {
		configured, err := resolveConfiguredString("email", "subject")
		if err != nil {
			return "", err
		}
		template = configured
	}
	if strings.TrimSpace(template) == "" {
		template = defaultEmailSubjectTemplate
	}

	subject := strings.TrimSpace(renderEmailTemplate(template, ctx))
	if subject == "" {
		return "", fmt.Errorf("%s: email subject resolved to empty value", invoicePath)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return "", fmt.Errorf("%s: email subject must be a single line", invoicePath)
	}
	return subject, nil
}

func invoiceEmailBody(ctx *Context) (string, error) {
	body, err := invoiceEmailBodyText(ctx)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(body, "\n", "\r\n"), nil
}

func invoiceEmailBodyText(ctx *Context) (string, error) {
	template, err := resolveConfiguredString("email", "body")
	if err != nil {
		return "", err
	}
	if template == "" {
		template = defaultEmailBodyTemplate
	}

	body := renderEmailTemplate(template, ctx)
	body = strings.TrimRight(body, "\n") + "\n\n"
	return body, nil
}

func renderEmailTemplate(template string, ctx *Context) string {
	template = strings.ReplaceAll(template, "\r\n", "\n")
	template = strings.ReplaceAll(template, "\r", "\n")
	return emailTemplateReplacer(ctx).Replace(template)
}

func emailTemplateReplacer(ctx *Context) *strings.Replacer {
	return strings.NewReplacer(
		"{customer_name}", customerName(ctx.Customer),
		"{email_greeting}", customerEmailGreeting(ctx.Customer),
		"{contact_person}", customerContactPerson(ctx.Customer),
		"{customer_id}", ctx.CustomerID,
		"{invoice_number}", ctx.InvoiceNumber,
		"{issue_date}", asString(ctx.Invoice["issue_date"]),
		"{due_date}", asString(ctx.Invoice["due_date"]),
		"{total_amount}", emailMoney(ctx.TotalCents, ctx.Currency),
		"{outstanding_amount}", emailMoney(ctx.OutstandingCents, ctx.Currency),
		"{payment_terms_text}", strings.TrimSpace(asString(ctx.IssuerPayment["payment_terms_text"])),
		"{issuer_name}", strings.TrimSpace(asString(ctx.IssuerCompany["legal_company_name"])),
	)
}

func emailMoney(cents int64, currency string) string {
	return formatMoneyCents(cents) + " " + currency
}

func writeBase64MIME(buffer io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := buffer.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if encoded != "" {
		if _, err := buffer.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}
