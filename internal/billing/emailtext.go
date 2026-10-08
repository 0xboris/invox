package billing

import (
	"errors"
	"strings"
)

const (
	// defaultSubjectTemplate is the subject when the config sets none.
	defaultSubjectTemplate = "Invoice {invoice_number}"
	// defaultBodyTemplate is the body when the config sets none.
	defaultBodyTemplate = `{email_greeting}

Please find attached invoice {invoice_number}.
Issue date: {issue_date}
Due date: {due_date}
Outstanding amount: {outstanding_amount}

Regards,
{issuer_name}
`
)

// EmailFields are the values the subject and body placeholders stand for, as
// they appear in the email.
type EmailFields struct {
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

// subject renders the subject template, or defaultSubjectTemplate when
// template is blank, and trims it. The result must be one non-empty line.
func subject(template string, f EmailFields) (string, error) {
	if strings.TrimSpace(template) == "" {
		template = defaultSubjectTemplate
	}
	subject := strings.TrimSpace(render(template, f))
	if subject == "" {
		return "", ErrEmptySubject
	}
	if strings.ContainsAny(subject, "\r\n") {
		return "", ErrMultilineSubject
	}
	return subject, nil
}

// body renders the body template, or defaultBodyTemplate when template is
// empty, and ends it with exactly one blank line.
func body(template string, f EmailFields) string {
	if template == "" {
		template = defaultBodyTemplate
	}
	body := render(template, f)
	return strings.TrimRight(body, "\n") + "\n\n"
}

// render replaces every placeholder in template with its field. Line
// endings become \n.
func render(template string, f EmailFields) string {
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
