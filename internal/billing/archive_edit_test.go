package billing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
)

func TestEditArchivedInvoiceRejectsLegacyKeys(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-06.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: archived
  period_label: March 2026
  vat_rate_percent: 20
  paid_amount: 0
line_items:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	workDir := t.TempDir()
	_, err := h.service(t, cmdutil.Files{}, workDir, time.Time{}).EditArchived("2026-03-06.yaml", workDir, billing.EditOptions{})
	if err == nil {
		t.Fatal("EditArchivedInvoice returned nil error for legacy keys")
	}
	for _, want := range []string{
		archivePath + ":7: invoice.period_label: unsupported key; use invoice.period",
		archivePath + ":8: invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		archivePath + ":10: line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}
