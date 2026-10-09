package billing_test

import (
	"context"
	"mime"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
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

	svc, opened := h.mailService(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now())
	draft, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: outputPath, Keep: true})
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.Draft != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.Draft, outputPath)
	}
	if want := []string{outputPath}; !reflect.DeepEqual(*opened, want) {
		t.Fatalf("opened %q, want %q", *opened, want)
	}
	if draft.Message.To != "office@appsters.example" {
		t.Fatalf("Recipient = %q, want %q", draft.Message.To, "office@appsters.example")
	}
	if draft.Message.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", draft.Message.Subject, "Invoice CUST-001-001")
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

	svc, _ := h.mailService(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now())
	draft, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: outputPath, Keep: true})
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.Draft != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.Draft, outputPath)
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

	svc, _ := h.mailService(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now())
	if _, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: outputPath, Keep: true}); err != nil {
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

	svc, _ := h.mailService(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now())
	draft, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: outputPath, Keep: true})
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	wantSubject := "Appsters GmbH | Dear Jane Doe, | Jane Doe | CUST-001 | CUST-001-001 | 2026-03-06 | 2026-04-05 | 252,00 EUR | 252,00 EUR | Pay within 30 days | Boris Consulting"
	if draft.Message.Subject != wantSubject {
		t.Fatalf("Subject = %q, want %q", draft.Message.Subject, wantSubject)
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

	svc, _ := h.mailService(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now())
	draft, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: outputPath, Keep: true, Subject: "Invoice {invoice_number} for {contact_person}"})
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	if draft.Message.Subject != "Invoice CUST-001-001 for Jane Doe" {
		t.Fatalf("Subject = %q, want %q", draft.Message.Subject, "Invoice CUST-001-001 for Jane Doe")
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

	_, err := h.service(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Now()).DraftEmail(context.Background(), billing.EmailRequest{Invoice: invoicePath, PDF: pdfPath, Output: filepath.Join(t.TempDir(), "invoice.eml"), Keep: true})
	if err == nil {
		t.Fatal("CreateInvoiceEmailDraft returned nil error for non-built invoice")
	}
	if !strings.Contains(err.Error(), "invoice.status must be `built` or `archived` before creating an email draft") {
		t.Fatalf("error %q does not contain sendable status validation", err.Error())
	}
}

func TestResolveEmailDraftPathsIgnoresAMatchOnlyInArchiveHistory(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	historyPath := filepath.Join(archiveDir, ".history", "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(historyPath)) returned error: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(historyPath) returned error: %v", err)
	}

	customersPath, issuerPath, _, _, _, _ := writeContextFixtures(t)
	pdfPath := filepath.Join(t.TempDir(), "BL00210001.pdf")
	result, err := h.service(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Time{}).DraftEmail(context.Background(), billing.EmailRequest{FromPDF: pdfPath, PDF: pdfPath, DryRun: true})
	want := pdfPath + ": no matching invoice YAML found next to the PDF or in archive.dir"
	if err == nil || err.Error() != want {
		t.Fatalf("ResolveEmailDraftPaths = %q, %v; want error %q", result.Number, err, want)
	}
}
