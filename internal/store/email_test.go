package store

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateInvoiceEmailDraftIncludesAttachmentAndHeaders(t *testing.T) {
	t.Parallel()

	h := testHost(filepath.Join(t.TempDir(), "config-home"), filepath.Join(t.TempDir(), "home"))

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, false)
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.OutputPath != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.OutputPath, outputPath)
	}
	if draft.Recipient != "office@appsters.example" {
		t.Fatalf("Recipient = %q, want %q", draft.Recipient, "office@appsters.example")
	}
	if draft.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", draft.Subject, "Invoice CUST-001-001")
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(eml)
	for _, want := range []string{
		"hello@example.com",
		"office@appsters.example",
		"Subject: Invoice CUST-001-001",
		"X-Unsent: 1",
		`filename="invoice.pdf"`,
		"Dear Jane Doe,",
		"Please find attached invoice CUST-001-001.",
		"Outstanding amount: 252,00 EUR",
		"Regards,\r\nBoris Consulting\r\n\r\n\r\n--invox-boundary-",
		"JVBERi0xLjQKZmFrZQ==",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("draft email does not contain %q:\n%s", want, text)
		}
	}
}

func TestCreateInvoiceEmailDraftAllowsArchivedInvoiceStatus(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: archived", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, false)
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.OutputPath != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.OutputPath, outputPath)
	}
}

func TestCreateInvoiceEmailDraftUsesConfiguredBodyTemplate(t *testing.T) {
	t.Parallel()

	h := writeConfigFile(t, strings.TrimSpace(`
email:
  body: |
    {email_greeting}

    Invoice {invoice_number} is due on {due_date}.
    Open amount: {outstanding_amount}
    Terms: {payment_terms_text}

    Regards,
    {issuer_name}
`)+"\n")

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	if _, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, false); err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(eml)
	for _, want := range []string{
		"Dear Jane Doe,",
		"Invoice CUST-001-001 is due on 2026-04-05.",
		"Open amount: 252,00 EUR",
		"Terms: Pay within 30 days",
		"Regards,",
		"Regards,\r\nBoris Consulting\r\n\r\n\r\n--invox-boundary-",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("draft email does not contain %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Please find attached invoice CUST-001-001.") {
		t.Fatalf("draft email should not contain the default body:\n%s", text)
	}
}

func TestCreateInvoiceEmailDraftUsesConfiguredSubjectTemplateWithAllPlaceholders(t *testing.T) {
	t.Parallel()

	h := writeConfigFile(t, strings.TrimSpace(`
email:
  subject: "{customer_name} | {email_greeting} | {contact_person} | {customer_id} | {invoice_number} | {issue_date} | {due_date} | {total_amount} | {outstanding_amount} | {payment_terms_text} | {issuer_name}"
`)+"\n")

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, false)
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	wantSubject := "Appsters GmbH | Dear Jane Doe, | Jane Doe | CUST-001 | CUST-001-001 | 2026-03-06 | 2026-04-05 | 252,00 EUR | 252,00 EUR | Pay within 30 days | Boris Consulting"
	if draft.Subject != wantSubject {
		t.Fatalf("Subject = %q, want %q", draft.Subject, wantSubject)
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	wantHeader := "Subject: " + mime.QEncoding.Encode("utf-8", wantSubject)
	if !strings.Contains(string(eml), wantHeader) {
		t.Fatalf("draft email does not contain %q:\n%s", wantHeader, string(eml))
	}
}

func TestCreateInvoiceEmailDraftExpandsSubjectOverridePlaceholders(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath, Subject: "Invoice {invoice_number} for {contact_person}"}, outputPath, false)
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	if draft.Subject != "Invoice CUST-001-001 for Jane Doe" {
		t.Fatalf("Subject = %q, want %q", draft.Subject, "Invoice CUST-001-001 for Jane Doe")
	}
}

func TestCreateInvoiceEmailDraftRejectsInvoiceWithoutSendableStatus(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	_, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, filepath.Join(t.TempDir(), "invoice.eml"), false)
	if err == nil {
		t.Fatal("CreateInvoiceEmailDraft returned nil error for non-built invoice")
	}
	if !strings.Contains(err.Error(), "invoice.status must be `built` or `archived` before creating an email draft") {
		t.Fatalf("error %q does not contain sendable status validation", err.Error())
	}
}
