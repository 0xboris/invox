package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeNumberedInvoice(t *testing.T, dir, name, invoiceNumber, status string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	source := strings.Join([]string{
		"customer_id: CUST-001",
		"invoice:",
		"  number: " + invoiceNumber,
		"  issue_date: \"2026-03-06\"",
		"  due_date: \"2026-04-05\"",
		"  status: " + status,
		"  vat_percent: 20",
		"  paid_amount: 0",
		"positions:",
		"  - name: Development",
		"    unit_price: 100",
		"    quantity: 1",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
	return path
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return string(source)
}
