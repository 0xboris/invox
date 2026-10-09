package billing_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
)

// markdownArchive is an archive with one YAML invoice, CUST-001-001, and
// a Markdown one with the higher number CUST-001-015, which invox no
// longer reads.
func markdownArchive(t *testing.T) (h host, archiveDir, markdownPath string) {
	t.Helper()
	archiveDir = t.TempDir()
	h = writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")
	markdownPath = filepath.Join(archiveDir, "old.md")
	source := "---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-015\n  issue_date: \"2026-03-05\"\n  status: archived\n---\n# Invoice\n"
	if err := os.WriteFile(markdownPath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", markdownPath, err)
	}
	return h, archiveDir, markdownPath
}

func TestArchiveWalksReportMarkdownInvoicesAsUnread(t *testing.T) {
	t.Parallel()

	h, archiveDir, markdownPath := markdownArchive(t)
	want := billing.Unread{Dir: archiveDir, Markdown: []string{markdownPath}}
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	files := cmdutil.Files{Customers: customersPath, Issuer: issuerPath, Defaults: defaultsPath}
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	workDir := t.TempDir()
	created, err := h.service(t, files, workDir, now).New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: workDir, DryRun: true})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if created.Number != "CUST-001-002" || !reflect.DeepEqual(created.Unread, want) {
		t.Fatalf("New = number %q, unread %+v; want CUST-001-002 and %+v", created.Number, created.Unread, want)
	}

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-001", "built")
	incremented, err := h.service(t, files, filepath.Dir(invoicePath), now).Increment(invoicePath, true)
	if err != nil {
		t.Fatalf("Increment returned error: %v", err)
	}
	if incremented.NewNumber != "CUST-001-002" || !reflect.DeepEqual(incremented.Unread, want) {
		t.Fatalf("Increment = number %q, unread %+v; want CUST-001-002 and %+v", incremented.NewNumber, incremented.Unread, want)
	}

	// The Markdown invoice's number is free to use.
	writeStatusInvoice(t, invoicePath, "CUST-001-015", "built")
	archived, err := h.service(t, files, filepath.Dir(invoicePath), now).Archive(invoicePath, billing.ArchiveOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}
	if !reflect.DeepEqual(archived.Unread, want) {
		t.Fatalf("Archive unread = %+v, want %+v", archived.Unread, want)
	}

	list, err := h.service(t, files, t.TempDir(), now).ListArchive()
	if err != nil {
		t.Fatalf("ListArchive returned error: %v", err)
	}
	if len(list.Entries) != 1 || list.Entries[0].Filename != "first.yaml" || !reflect.DeepEqual(list.Unread, want) {
		t.Fatalf("ListArchive = %+v, want only first.yaml and unread %+v", list, want)
	}

	pdf := filepath.Join(t.TempDir(), "old.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = h.service(t, files, t.TempDir(), now).DraftEmail(context.Background(), billing.EmailRequest{FromPDF: pdf, PDF: pdf, DryRun: true})
	if want := pdf + ": no matching invoice YAML found next to the PDF or in archive.dir"; err == nil || err.Error() != want {
		t.Fatalf("DraftEmail error = %v, want %q", err, want)
	}
}

func TestEditArchivedRefusesAMarkdownInvoice(t *testing.T) {
	t.Parallel()

	h, _, markdownPath := markdownArchive(t)
	workDir := t.TempDir()
	_, err := h.service(t, cmdutil.Files{}, workDir, time.Time{}).EditArchived("old.md", workDir, billing.EditOptions{})
	if want := markdownPath + " is a Markdown invoice, which invox no longer reads; convert it to .yaml"; err == nil || err.Error() != want {
		t.Fatalf("EditArchived(old.md) error = %v, want %q", err, want)
	}
	if entries, _ := os.ReadDir(workDir); len(entries) != 0 {
		t.Fatalf("EditArchived wrote %v, want nothing", entries)
	}
}
