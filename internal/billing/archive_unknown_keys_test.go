package billing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/testfixture"
)

// An archived invoice is a record: `new --from-last` copies keys invox does
// not know as they are, and `archive edit` opens such an invoice. Only the
// defaults file is held to the schema when `new` reads it.
func TestNewAndArchiveEditKeepUnknownKeysOfArchivedInvoices(t *testing.T) {
	t.Parallel()

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
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
	workDir := t.TempDir()
	if _, err := service(t, h, cmdutil.Files{Customers: draft.Customers, Issuer: draft.Issuer, Defaults: draft.Defaults}, workDir, time.Now()).New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: workDir, Output: outputPath, FromLast: true}); err != nil {
		t.Fatalf("CreateNewInvoice --from-last returned error: %v", err)
	}
	if created := testfixture.ReadFile(t, outputPath); !strings.Contains(created, "\nnotes: call before invoicing\n") {
		t.Fatalf("new invoice does not keep notes:\n%s", created)
	}

	workDir = t.TempDir()
	opened, err := service(t, h, cmdutil.Files{}, workDir, time.Time{}).EditArchived("2026-03-08.yaml", workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	workingCopy := opened.Path
	if edited := testfixture.ReadFile(t, workingCopy); !strings.Contains(edited, "\nnotes: call before invoicing\n") {
		t.Fatalf("working copy does not keep notes:\n%s", edited)
	}

	replaceInFixture(t, draft.Defaults, "invoice:\n", "notes: x\ninvoice:\n")
	workDir = t.TempDir()
	_, err = service(t, h, cmdutil.Files{Customers: draft.Customers, Issuer: draft.Issuer, Defaults: draft.Defaults}, workDir, time.Now()).New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: workDir, Output: filepath.Join(t.TempDir(), "next.yaml")})
	if want := draft.Defaults + `:1: unknown key "notes"`; err == nil || err.Error() != want {
		t.Fatalf("CreateNewInvoice from defaults error = %v, want %q", err, want)
	}
}

// The keys old versions of invox wrote are unknown keys like any other: an
// archived invoice that has them still opens, the copy keeps them, and the
// copy fails validation, naming each one.
func TestLegacyKeysOfArchivedInvoicesAreUnknownKeys(t *testing.T) {
	t.Parallel()

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	archivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period_label: March 2026
  vat_rate_percent: 10
  paid_amount: 0
line_items:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := cmdutil.Files{Customers: draft.Customers, Issuer: draft.Issuer, Defaults: draft.Defaults}

	workDir := t.TempDir()
	created, err := service(t, h, files, workDir, time.Now()).New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: workDir, Output: filepath.Join(workDir, "next.yaml"), FromLast: true})
	if err != nil {
		t.Fatalf("New --from-last returned error: %v", err)
	}
	_, err = service(t, h, files, workDir, time.Time{}).Validate(created.Path)
	if want := created.Path + `:7: unknown key "period_label" in invoice` + "\n" +
		created.Path + `:8: unknown key "vat_rate_percent" in invoice` + "\n" +
		created.Path + `:10: unknown key "line_items"`; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Validate of the new invoice: error = %v, want it to start with %q", err, want)
	}

	workDir = t.TempDir()
	opened, err := service(t, h, files, workDir, time.Time{}).EditArchived("2026-03-08.yaml", workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchived returned error: %v", err)
	}
	_, err = service(t, h, files, workDir, time.Time{}).Validate(opened.Path)
	if want := opened.Path + `:7: unknown key "period_label" in invoice`; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Validate of the working copy: error = %v, want it to start with %q", err, want)
	}
}
