package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			write: MarkInvoiceBuilt,
			want:  strings.Replace(commentedInvoice, "status: draft #", "status: built #", 1),
		},
		{
			name:  "SetInvoiceStatus",
			write: func(path string) error { return SetInvoiceStatus(path, "archived") },
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

// Legacy Markdown archives go through the same decoder as YAML ones, with
// line numbers of the Markdown file.
func TestArchivedMarkdownInvoiceUsesTheInvoiceDecoder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "2025-01.md")
	source := "---\ncustomer_id: 0042\nnotes: kept by older versions\ninvoice:\n  number: 0042-007\n  issue_date: 2025-1-5\n  status: archived\n---\n# Invoice\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	identity, ok, err := archivedInvoiceIdentity(path)
	if err != nil || !ok {
		t.Fatalf("archivedInvoiceIdentity = ok %v, error %v; want an invoice", ok, err)
	}
	got := []string{identity.CustomerID.Trim(), identity.Invoice.Number.Trim(), identity.Invoice.IssueDate.Trim(), identity.Invoice.Status.Trim()}
	if want := []string{"0042", "0042-007", "2025-01-05", "archived"}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("identity = %q, want %q", got, want)
	}

	document, ok, err := loadArchivedInvoiceDocument(path)
	if err != nil || !ok {
		t.Fatalf("loadArchivedInvoiceDocument = ok %v, error %v", ok, err)
	}
	err = decodeYAMLDocument(document, path, &InvoiceFile{}, true)
	if want := path + `:3: unknown key "notes"`; err == nil || err.Error() != want {
		t.Fatalf("strict decode error = %v, want %q", err, want)
	}
}
