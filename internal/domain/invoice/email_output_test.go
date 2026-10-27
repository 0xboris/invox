package invoice

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateInvoiceEmailDraftRefusesExistingOutputUnlessOverwrite(t *testing.T) {
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
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	_, err = createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, false)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("CreateInvoiceEmailDraft error = %v, want fs.ErrExist", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if string(content) != "keep\n" {
		t.Fatalf("outputPath content = %q, want it untouched", content)
	}

	if _, err := createEmailDraft(h, time.Now(), EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath}, outputPath, true); err != nil {
		t.Fatalf("CreateInvoiceEmailDraft with overwrite returned error: %v", err)
	}
	content, err = os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if !strings.Contains(string(content), "X-Unsent: 1") {
		t.Fatalf("outputPath was not overwritten with the draft:\n%s", content)
	}
}
