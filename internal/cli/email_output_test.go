package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeBuiltEmailFixture writes a built invoice YAML and its PDF into a fresh directory and
// returns the customers, issuer and invoice paths.
func writeBuiltEmailFixture(t *testing.T) (string, string, string) {
	t.Helper()

	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)

	inputDir := t.TempDir()
	builtInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(builtInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(builtInvoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "BL00210001.pdf"), []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	return customersPath, issuerPath, builtInvoicePath
}
