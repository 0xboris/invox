// Package email renders the subject and body of an invoice email from
// their templates and builds the .eml draft that carries the PDF. It knows
// the placeholders and the MIME layout, not invoices: the caller fills in
// Fields.
package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultSubjectTemplate is the subject when the config sets none.
	DefaultSubjectTemplate = "Invoice {invoice_number}"
	// DefaultBodyTemplate is the body when the config sets none.
	DefaultBodyTemplate = `{email_greeting}

Please find attached invoice {invoice_number}.
Issue date: {issue_date}
Due date: {due_date}
Outstanding amount: {outstanding_amount}

Regards,
{issuer_name}
`
)

// Fields are the values the subject and body placeholders stand for, as
// they appear in the email.
type Fields struct {
	CustomerName      string // {customer_name}
	Greeting          string // {email_greeting}
	ContactPerson     string // {contact_person}
	CustomerID        string // {customer_id}
	InvoiceNumber     string // {invoice_number}
	IssueDate         string // {issue_date}
	DueDate           string // {due_date}
	TotalAmount       string // {total_amount}
	OutstandingAmount string // {outstanding_amount}
	PaymentTermsText  string // {payment_terms_text}
	IssuerName        string // {issuer_name}
}

// ErrEmptySubject reports a subject template that rendered to nothing.
var ErrEmptySubject = errors.New("email subject resolved to empty value")

// ErrMultilineSubject reports a subject that rendered to more than one line.
var ErrMultilineSubject = errors.New("email subject must be a single line")

// Subject renders the subject template, or DefaultSubjectTemplate when
// template is blank, and trims it. The result must be one non-empty line.
func Subject(template string, f Fields) (string, error) {
	if strings.TrimSpace(template) == "" {
		template = DefaultSubjectTemplate
	}
	subject := strings.TrimSpace(Render(template, f))
	if subject == "" {
		return "", ErrEmptySubject
	}
	if strings.ContainsAny(subject, "\r\n") {
		return "", ErrMultilineSubject
	}
	return subject, nil
}

// Body renders the body template, or DefaultBodyTemplate when template is
// empty, and ends it with exactly one blank line.
func Body(template string, f Fields) string {
	if template == "" {
		template = DefaultBodyTemplate
	}
	body := Render(template, f)
	return strings.TrimRight(body, "\n") + "\n\n"
}

// Render replaces every placeholder in template with its field. Line
// endings become \n.
func Render(template string, f Fields) string {
	template = strings.ReplaceAll(template, "\r\n", "\n")
	template = strings.ReplaceAll(template, "\r", "\n")
	return strings.NewReplacer(
		"{customer_name}", f.CustomerName,
		"{email_greeting}", f.Greeting,
		"{contact_person}", f.ContactPerson,
		"{customer_id}", f.CustomerID,
		"{invoice_number}", f.InvoiceNumber,
		"{issue_date}", f.IssueDate,
		"{due_date}", f.DueDate,
		"{total_amount}", f.TotalAmount,
		"{outstanding_amount}", f.OutstandingAmount,
		"{payment_terms_text}", f.PaymentTermsText,
		"{issuer_name}", f.IssuerName,
	).Replace(template)
}

// Draft is an email with one PDF attachment, ready to write as .eml.
type Draft struct {
	Recipient     string
	Subject       string
	Body          string
	SenderName    string
	SenderAddress string
	// AttachmentPath names the PDF; only its base name goes into the draft.
	AttachmentPath string
}

// Build encodes d as a multipart/mixed message: the body quoted-printable,
// pdf base64 under the attachment's base name, dated now, marked unsent so
// mail clients open it for editing. boundary separates the parts.
func Build(d Draft, pdf []byte, now time.Time, boundary string) ([]byte, error) {
	fromAddress := mail.Address{
		Name:    d.SenderName,
		Address: d.SenderAddress,
	}
	toAddress := mail.Address{
		Address: d.Recipient,
	}

	var buffer bytes.Buffer

	fmt.Fprintf(&buffer, "From: %s\r\n", fromAddress.String())
	fmt.Fprintf(&buffer, "To: %s\r\n", toAddress.String())
	fmt.Fprintf(&buffer, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", d.Subject))
	fmt.Fprintf(&buffer, "Date: %s\r\n", now.Format(time.RFC1123Z))
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
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return nil, err
	}
	bodyWriter := quotedprintable.NewWriter(textPart)
	if _, err := bodyWriter.Write([]byte(d.Body)); err != nil {
		return nil, err
	}
	if err := bodyWriter.Close(); err != nil {
		return nil, err
	}

	filename := filepath.Base(d.AttachmentPath)
	attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {formatMIMEParameter("application/pdf", "name", filename)},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {formatMIMEParameter("attachment", "filename", filename)},
	})
	if err != nil {
		return nil, err
	}
	if err := writeBase64MIME(attachmentPart, pdf); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// formatMIMEParameter keeps the plain quoted form for printable ASCII values
// without quotes or backslashes, and otherwise lets mime.FormatMediaType escape
// the value or RFC 2231-encode it.
func formatMIMEParameter(mediaType, key, value string) string {
	plain := true
	for i := 0; i < len(value); i++ {
		if c := value[i]; c < 0x20 || c > 0x7e || c == '"' || c == '\\' {
			plain = false
			break
		}
	}
	if plain {
		return fmt.Sprintf(`%s; %s="%s"`, mediaType, key, value)
	}
	return mime.FormatMediaType(mediaType, map[string]string{key: value})
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
