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

func TestEditArchivedMarkdownInvoiceAndRearchiveAsYAML(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedMarkdownPath := filepath.Join(archiveDir, "customer-a", "2026-03-06.md")
	if err := os.MkdirAll(filepath.Dir(archivedMarkdownPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive subdir) returned error: %v", err)
	}
	if err := os.WriteFile(archivedMarkdownPath, []byte(strings.Join([]string{
		"---",
		"customer_id: CUST-001",
		"invoice:",
		"  number: CUST-001-001",
		"  issue_date: 2026-03-06",
		"  due_date: 2026-04-05",
		"  status: archived",
		"  period: March 2026",
		"  vat_percent: 20",
		"  paid_amount: 0",
		"positions:",
		"  - name: Development",
		"    description: Markdown archive",
		"    unit_price: 100",
		"    quantity: 2",
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedMarkdownPath) returned error: %v", err)
	}

	workDir := t.TempDir()
	opened, err := h.service(t, cmdutil.Files{}, workDir, time.Time{}).EditArchived("customer-a/2026-03-06.md", workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	outputPath, archivePath := opened.Path, opened.Archived
	if archivePath != archivedMarkdownPath {
		t.Fatalf("archivePath = %q, want %q", archivePath, archivedMarkdownPath)
	}

	wantOutputPath := filepath.Join(workDir, "2026-03-06.yaml")
	if outputPath != wantOutputPath {
		t.Fatalf("outputPath = %q, want %q", outputPath, wantOutputPath)
	}
	editedSource, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	editedText := string(editedSource)
	for _, want := range []string{
		"status: editing",
		"archive_path: customer-a/2026-03-06.yaml",
		"archive_replace_path: customer-a/2026-03-06.md",
		"Markdown archive",
	} {
		if !strings.Contains(editedText, want) {
			t.Fatalf("edited invoice does not contain %q:\n%s", want, editedText)
		}
	}

	mutated := strings.Replace(editedText, "Markdown archive", "Edited markdown archive", 1)
	if err := os.WriteFile(outputPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	result, err := h.service(t, cmdutil.Files{}, filepath.Dir(outputPath), time.Now()).Archive(outputPath, billing.ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("ArchiveInvoice returned error: %v", err)
	}
	finalArchivePath := result.Path
	wantArchivePath := filepath.Join(archiveDir, "customer-a", "2026-03-06.yaml")
	if finalArchivePath != wantArchivePath {
		t.Fatalf("finalArchivePath = %q, want %q", finalArchivePath, wantArchivePath)
	}
	if _, err := os.Stat(archivedMarkdownPath); err == nil {
		t.Fatalf("legacy markdown archive should have been removed: %s", archivedMarkdownPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(archivedMarkdownPath) returned unexpected error: %v", err)
	}
	if _, err := os.Stat(outputPath); err == nil {
		t.Fatalf("working copy should have been removed: %s", outputPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(outputPath) returned unexpected error: %v", err)
	}

	finalSource, err := os.ReadFile(finalArchivePath)
	if err != nil {
		t.Fatalf("ReadFile(finalArchivePath) returned error: %v", err)
	}
	finalText := string(finalSource)
	for _, want := range []string{
		"status: archived",
		"Edited markdown archive",
	} {
		if !strings.Contains(finalText, want) {
			t.Fatalf("final archive does not contain %q:\n%s", want, finalText)
		}
	}
	for _, forbidden := range []string{
		"status: editing",
		"_invox:",
	} {
		if strings.Contains(finalText, forbidden) {
			t.Fatalf("final archive should not contain %q:\n%s", forbidden, finalText)
		}
	}
}

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
