package billing

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/money"
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

// EmailPlaceholder is a placeholder of the email subject and body.
type EmailPlaceholder struct {
	// Name is the placeholder as written, such as {invoice_number}.
	Name        string
	Description string
	value       func(*invoice.Context) string
}

var emailPlaceholders = []EmailPlaceholder{
	{"{customer_name}", "Customer display name", func(c *invoice.Context) string { return c.Customer.DisplayName() }},
	{"{email_greeting}", "Customer-specific greeting, defaults to Hello,", func(c *invoice.Context) string { return c.Customer.Greeting() }},
	{"{contact_person}", "Customer contact person", func(c *invoice.Context) string { return c.Customer.Contact() }},
	{"{customer_id}", "Customer ID from the invoice", func(c *invoice.Context) string { return c.CustomerID }},
	{"{invoice_number}", "Invoice number", func(c *invoice.Context) string { return c.InvoiceNumber }},
	{"{issue_date}", "Invoice issue date", func(c *invoice.Context) string { return c.Header.IssueDate.String() }},
	{"{due_date}", "Invoice due date", func(c *invoice.Context) string { return c.Header.DueDate.String() }},
	{"{total_amount}", "Invoice total with currency", func(c *invoice.Context) string { return emailMoney(c.TotalCents, c.Currency) }},
	{"{outstanding_amount}", "Outstanding amount with currency", func(c *invoice.Context) string {
		return emailMoney(c.OutstandingCents, c.Currency)
	}},
	{"{payment_terms_text}", "issuer.payment.payment_terms_text", func(c *invoice.Context) string { return c.Payment.PaymentTermsText.Trim() }},
	{"{issuer_name}", "issuer.company.legal_company_name", func(c *invoice.Context) string { return c.Company.LegalCompanyName.Trim() }},
}

// EmailPlaceholders returns the placeholders of email.subject and
// email.body, in the order help lists them.
func EmailPlaceholders() []EmailPlaceholder {
	return slices.Clone(emailPlaceholders)
}

func emailMoney(cents int64, currency string) string {
	return money.FormatCents(cents) + " " + currency
}

// emailPlaceholderPattern matches every token written like a placeholder.
var emailPlaceholderPattern = regexp.MustCompile(`\{[a-z_]+\}`)

// checkEmailTemplates reports every placeholder of the subject and body
// templates that emailPlaceholders lacks, one per line.
func checkEmailTemplates(subject, body string) error {
	return errors.Join(unknownEmailPlaceholders("email.subject", subject), unknownEmailPlaceholders("email.body", body))
}

func unknownEmailPlaceholders(setting, template string) error {
	var errs []error
	var seen []string
	for _, name := range emailPlaceholderPattern.FindAllString(template, -1) {
		known := slices.ContainsFunc(emailPlaceholders, func(p EmailPlaceholder) bool { return p.Name == name })
		if !known && !slices.Contains(seen, name) {
			seen = append(seen, name)
			errs = append(errs, fmt.Errorf("%s: unknown placeholder %s", setting, name))
		}
	}
	return errors.Join(errs...)
}

// emailValues replaces every placeholder with its value for ctx.
func emailValues(ctx *invoice.Context) *strings.Replacer {
	pairs := make([]string, 0, 2*len(emailPlaceholders))
	for _, p := range emailPlaceholders {
		pairs = append(pairs, p.Name, p.value(ctx))
	}
	return strings.NewReplacer(pairs...)
}

// ErrEmptySubject reports a subject template that rendered to nothing.
var ErrEmptySubject = errors.New("email subject resolved to empty value")

// ErrMultilineSubject reports a subject that rendered to more than one line.
var ErrMultilineSubject = errors.New("email subject must be a single line")

// subject renders the subject template, or defaultSubjectTemplate when
// template is blank, and trims it. The result must be one non-empty line.
func subject(template string, values *strings.Replacer) (string, error) {
	if strings.TrimSpace(template) == "" {
		template = defaultSubjectTemplate
	}
	subject := strings.TrimSpace(render(template, values))
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
func body(template string, values *strings.Replacer) string {
	if template == "" {
		template = defaultBodyTemplate
	}
	body := render(template, values)
	return strings.TrimRight(body, "\n") + "\n\n"
}

// render replaces every placeholder in template with its value. Line
// endings become \n.
func render(template string, values *strings.Replacer) string {
	template = strings.ReplaceAll(template, "\r\n", "\n")
	template = strings.ReplaceAll(template, "\r", "\n")
	return values.Replace(template)
}
