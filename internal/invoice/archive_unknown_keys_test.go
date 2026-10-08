package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// An archived invoice is a record: `new --from-last` copies keys invox does
// not know as they are, and `archive edit` opens such an invoice. Only the
// defaults file is held to the schema when `new` reads it.
func TestNewAndArchiveEditKeepUnknownKeysOfArchivedInvoices(t *testing.T) {
	t.Parallel()

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
notes: call before invoicing
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period: March 2026
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	if _, err := h.CreateNewInvoice(time.Now(), t.TempDir(), defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", true); err != nil {
		t.Fatalf("CreateNewInvoice --from-last returned error: %v", err)
	}
	if created := readTestFile(t, outputPath); !strings.Contains(created, "\nnotes: call before invoicing\n") {
		t.Fatalf("new invoice does not keep notes:\n%s", created)
	}

	workingCopy, _, err := h.EditArchivedInvoice("2026-03-08.yaml", t.TempDir())
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if edited := readTestFile(t, workingCopy); !strings.Contains(edited, "\nnotes: call before invoicing\n") {
		t.Fatalf("working copy does not keep notes:\n%s", edited)
	}

	replaceInFixture(t, defaultsPath, "invoice:\n", "notes: x\ninvoice:\n")
	_, err = h.CreateNewInvoice(time.Now(), t.TempDir(), defaultsPath, filepath.Join(t.TempDir(), "next.yaml"), customersPath, issuerPath, "CUST-001", false)
	if want := defaultsPath + `:1: unknown key "notes"`; err == nil || err.Error() != want {
		t.Fatalf("CreateNewInvoice from defaults error = %v, want %q", err, want)
	}
}
