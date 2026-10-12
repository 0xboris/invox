package billing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMinimalInvoice(t *testing.T, invoiceNumber string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "invoice.yaml")
	source := strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: PLACEHOLDER
  issue_date: 2026-03-06
`) + "\n"
	source = strings.Replace(source, "PLACEHOLDER", invoiceNumber, 1)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	return path
}

func writeInvoiceWithoutNumber(t *testing.T, sourcePath string) string {
	t.Helper()

	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(invoice.yaml) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  number: CUST-001-001\n", "", 1)
	if mutated == string(source) {
		t.Fatal("failed to remove invoice.number from the fixture")
	}

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	return invoicePath
}

func replaceInFixture(t *testing.T, path, old, replacement string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	if !strings.Contains(string(source), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(source), old, replacement, 1)), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}
