package billing

import (
	"strings"
	"testing"
)

func FuzzRenderEmailTemplate(f *testing.F) {
	for _, seed := range []string{
		"",
		defaultBodyTemplate,
		"{customer_name} | {email_greeting} | {contact_person} | {customer_id} | {invoice_number} | {issue_date} | {due_date} | {total_amount} | {outstanding_amount} | {payment_terms_text} | {issuer_name}",
		"Invoice {invoice_number}\r\nTotal {total_amount}\r",
		"{{invoice_number}}",
		"{unknown}",
	} {
		f.Add(seed)
	}

	// The values contain no braces, so every placeholder is replaced and none
	// can be formed by a replacement.
	fields := EmailFields{
		CustomerName:      "Example GmbH",
		Greeting:          "Hello,",
		ContactPerson:     "Erika Mustermann",
		CustomerID:        "CUST-001",
		InvoiceNumber:     "CUST-001-007",
		IssueDate:         "2026-03-06",
		DueDate:           "2026-03-20",
		TotalAmount:       "1.234,56 EUR",
		OutstandingAmount: "1.234,56 EUR",
		PaymentTermsText:  "Payable within 14 days.",
		IssuerName:        "Issuer AG",
	}
	placeholders := []string{
		"{customer_name}", "{email_greeting}", "{contact_person}", "{customer_id}", "{invoice_number}",
		"{issue_date}", "{due_date}", "{total_amount}", "{outstanding_amount}", "{payment_terms_text}", "{issuer_name}",
	}
	f.Fuzz(func(t *testing.T, template string) {
		rendered := render(template, fields)
		if strings.Contains(rendered, "\r") {
			t.Fatalf("render(%q) = %q, which still contains a carriage return", template, rendered)
		}
		for _, placeholder := range placeholders {
			if strings.Contains(rendered, placeholder) {
				t.Fatalf("render(%q) = %q, which still contains %s", template, rendered, placeholder)
			}
		}
		if !strings.Contains(template, "{") {
			normalized := strings.ReplaceAll(strings.ReplaceAll(template, "\r\n", "\n"), "\r", "\n")
			if rendered != normalized {
				t.Fatalf("render(%q) = %q, want it unchanged", template, rendered)
			}
		}
	})
}
