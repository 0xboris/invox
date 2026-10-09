package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

const commentedInvoice = `# Invoice for ACME
customer_id: CUST-001 # the customer
invoice:
  # set by new
  number: CUST-001-001
  status: draft # draft until built
  period: March
positions:
  # first position
  - name: Dev
    unit_price: 100
# trailing note
`

// Status and number updates rewrite the file through yaml.Node, so every
// comment stays where it was.
func TestInvoiceWritesKeepComments(t *testing.T) {
	tests := []struct {
		name  string
		write func(path string) error
		want  string
	}{
		{
			name:  "MarkInvoiceBuilt",
			write: func(path string) error { return setInvoiceStatus(path, string(invoice.Built)) },
			want:  strings.Replace(commentedInvoice, "status: draft #", "status: built #", 1),
		},
		{
			name:  "SetInvoiceStatus",
			write: func(path string) error { return setInvoiceStatus(path, "archived") },
			want:  strings.Replace(commentedInvoice, "status: draft #", "status: archived #", 1),
		},
		{
			name:  "writeInvoiceNumber",
			write: func(path string) error { return writeInvoiceNumber(path, "CUST-001-002") },
			want:  strings.Replace(commentedInvoice, "number: CUST-001-001", "number: CUST-001-002", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invoice.yaml")
			if err := os.WriteFile(path, []byte(commentedInvoice), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := tt.write(path); err != nil {
				t.Fatalf("write returned error: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("file =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

// writeInvoiceNumber and setInvoiceStatus write one header field through
// Store.Update, as the use cases do.
func writeInvoiceNumber(path, invoiceNumber string) error {
	return (&Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Number = invoice.Text(invoiceNumber)
		return nil
	})
}

func setInvoiceStatus(path, status string) error {
	return (&Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Status = invoice.Text(status)
		return nil
	})
}
