package billing_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/render/latex"
	"github.com/0xboris/invox/internal/testfixture"
)

// archiveBackupTime is the time the replacement tests archive at, so the
// backup file names are known.
var archiveBackupTime = time.Date(2026, 10, 5, 12, 30, 45, 0, time.UTC)

// editArchivedForTest opens archiveName with `archive edit` and changes the
// working copy, so re-archiving it is a real replacement.
func editArchivedForTest(t *testing.T, h testfixture.Host, archiveName string) string {
	t.Helper()

	workDir := t.TempDir()
	opened, err := service(t, h, cmdutil.Files{}, workDir, time.Time{}).EditArchived(archiveName, workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	workingCopy := opened.Path
	// Add notes only once, so editing an invoice that was already edited
	// once still has a single notes key.
	source := testfixture.ReadFile(t, workingCopy)
	if !strings.Contains(source, "\nnotes: edited\n") {
		source += "notes: edited\n"
	}
	if err := os.WriteFile(workingCopy, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", workingCopy, err)
	}
	return workingCopy
}

func TestArchiveInvoiceRefusesToReplaceWithoutReplaceOption(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, archivedPath, "CUST-001-001", "archived")
	original := testfixture.ReadFile(t, archivedPath)
	workingCopy := editArchivedForTest(t, h, "first.yaml")

	_, err := service(t, h, cmdutil.Files{}, filepath.Dir(workingCopy), time.Now()).Archive(workingCopy, billing.ArchiveOptions{})
	var replaceErr *billing.ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		t.Fatalf("ArchiveInvoice error = %v, want *billing.ArchiveReplaceError", err)
	}
	if len(replaceErr.Paths) != 1 || replaceErr.Paths[0] != archivedPath {
		t.Fatalf("Paths = %q, want [%q]", replaceErr.Paths, archivedPath)
	}
	if want := filepath.Join(archiveDir, ".history"); replaceErr.HistoryDir != want {
		t.Fatalf("HistoryDir = %q, want %q", replaceErr.HistoryDir, want)
	}
	if got := testfixture.ReadFile(t, archivedPath); got != original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(workingCopy); err != nil {
		t.Fatalf("working copy should stay in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}
}

func TestArchiveInvoiceReplaceKeepsBackupInHistory(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, archivedPath, "CUST-001-001", "archived")
	original := testfixture.ReadFile(t, archivedPath)

	workingCopy := editArchivedForTest(t, h, "first.yaml")
	result, err := service(t, h, cmdutil.Files{}, filepath.Dir(workingCopy), archiveBackupTime).Archive(workingCopy, billing.ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("ArchiveInvoice returned error: %v", err)
	}
	backupPath := filepath.Join(archiveDir, ".history", "first.20261005T123045Z.yaml")
	if want := []billing.Backup{{Path: archivedPath, BackupPath: backupPath}}; len(result.Replaced) != 1 || result.Replaced[0] != want[0] {
		t.Fatalf("Replaced = %+v, want %+v", result.Replaced, want)
	}
	if got := testfixture.ReadFile(t, backupPath); got != original {
		t.Fatalf("backup = %q, want the previous version %q", got, original)
	}
	if !strings.Contains(testfixture.ReadFile(t, archivedPath), "notes: edited") {
		t.Fatalf("archived invoice was not replaced:\n%s", testfixture.ReadFile(t, archivedPath))
	}

	// A second replacement in the same second gets its own backup, and the
	// first backup does not count as a duplicate number.
	workingCopy = editArchivedForTest(t, h, "first.yaml")
	result, err = service(t, h, cmdutil.Files{}, filepath.Dir(workingCopy), archiveBackupTime).Archive(workingCopy, billing.ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("second ArchiveInvoice returned error: %v", err)
	}
	if want := filepath.Join(archiveDir, ".history", "first.20261005T123045Z-2.yaml"); len(result.Replaced) != 1 || result.Replaced[0].BackupPath != want {
		t.Fatalf("second Replaced = %+v, want backup %s", result.Replaced, want)
	}

	list, err := service(t, h, cmdutil.Files{}, t.TempDir(), time.Time{}).ListArchive()
	if err != nil {
		t.Fatalf("ListArchivedInvoices returned error: %v", err)
	}
	summaries := list.Entries
	if len(summaries) != 1 || summaries[0].Filename != "first.yaml" {
		t.Fatalf("ListArchivedInvoices = %+v, want only first.yaml", summaries)
	}
}

func TestArchiveInvoiceReplaceStillChecksDuplicateNumbers(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	firstPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, firstPath, "CUST-001-001", "archived")
	writeStatusInvoice(t, filepath.Join(archiveDir, "second.yaml"), "CUST-001-002", "archived")
	original := testfixture.ReadFile(t, firstPath)

	workingCopy := editArchivedForTest(t, h, "first.yaml")
	if err := setInvoiceNumber(workingCopy, "CUST-001-002"); err != nil {
		t.Fatalf("writeInvoiceNumber returned error: %v", err)
	}

	_, err := service(t, h, cmdutil.Files{}, filepath.Dir(workingCopy), time.Now()).Archive(workingCopy, billing.ArchiveOptions{Replace: true})
	var duplicate *invoice.DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	if got := testfixture.ReadFile(t, firstPath); got != original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}
}

func TestArchiveHistoryIsIgnoredByNumberingAndDuplicateCheck(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")
	historyDir := filepath.Join(archiveDir, ".history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(.history) returned error: %v", err)
	}
	writeStatusInvoice(t, filepath.Join(historyDir, "old.20261005T123045Z.yaml"), "CUST-001-009", "archived")

	draft := testfixture.WriteDraft(t)
	files := cmdutil.Files{Customers: draft.Customers, Issuer: draft.Issuer, Defaults: draft.Defaults}
	workDir := t.TempDir()
	created, err := service(t, h, files, workDir, time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)).New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: workDir, DryRun: true})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if created.Number != "CUST-001-002" {
		t.Fatalf("New number = %q, want CUST-001-002 (backups must not count)", created.Number)
	}

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-009", "built")
	if _, err := service(t, h, cmdutil.Files{}, filepath.Dir(invoicePath), time.Time{}).Archive(invoicePath, billing.ArchiveOptions{DryRun: true}); err != nil {
		t.Fatalf("Archive dry run = %v, want nil (backups must not count as duplicates)", err)
	}

	list, err := service(t, h, cmdutil.Files{}, t.TempDir(), time.Time{}).ListArchive()
	if err != nil {
		t.Fatalf("ListArchivedInvoices returned error: %v", err)
	}
	summaries := list.Entries
	if len(summaries) != 1 || summaries[0].Filename != "first.yaml" {
		t.Fatalf("ListArchivedInvoices = %+v, want only first.yaml", summaries)
	}
}

func TestMarkInvoiceBuiltKeepsArchivedStatus(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   string
	}{
		{status: "draft", want: "built"},
		{status: "editing", want: "built"},
		{status: "built", want: "built"},
		{status: "archived", want: "archived"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			fx := testfixture.WriteContext(t)
			replaceInFixture(t, fx.Invoice, "  paid_amount: 0\n", "  paid_amount: 0\n  status: "+tc.status+"\n")

			svc := service(t, testfixture.NewHost(t), cmdutil.Files{Customers: fx.Customers, Issuer: fx.Issuer}, t.TempDir(), time.Time{})
			renderer := svc.Renderer.(latex.Renderer)
			renderer.Compiler = writePDF{}
			svc.Renderer = renderer
			if _, err := svc.Build(context.Background(), billing.BuildRequest{Invoice: fx.Invoice, Template: fx.Template, Output: filepath.Join(t.TempDir(), "invoice.pdf")}); err != nil {
				t.Fatalf("MarkInvoiceBuilt returned error: %v", err)
			}
			if source := testfixture.ReadFile(t, fx.Invoice); !strings.Contains(source, "status: "+tc.want+"\n") {
				t.Fatalf("status = %s, want %s:\n%s", tc.status, tc.want, source)
			}
		})
	}
}

func TestArchiveHistoryPathsAreRefused(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := testfixture.HostWithConfig(t, "archive:\n  dir: "+testfixture.QuoteYAML(archiveDir)+"\n")
	historyDir := filepath.Join(archiveDir, ".history", "customer-a")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	writeStatusInvoice(t, filepath.Join(historyDir, "first.20261005T123045Z.yaml"), "CUST-001-001", "archived")

	for _, name := range []string{
		".history/customer-a/first.20261005T123045Z.yaml",
		"customer-a/../.history/customer-a/first.20261005T123045Z.yaml",
		".HISTORY/customer-a/first.20261005T123045Z.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			workDir := t.TempDir()
			_, err := service(t, h, cmdutil.Files{}, workDir, time.Time{}).EditArchived(name, workDir, billing.EditOptions{})
			if err == nil || !strings.Contains(err.Error(), "is a backup in .history, not an archived invoice") {
				t.Fatalf("EditArchivedInvoice error = %v, want backup refusal", err)
			}
		})
	}

	// A working copy whose archive_path points into .history is refused too,
	// so a backup cannot be overwritten by re-archiving.
	workingCopy := filepath.Join(t.TempDir(), "first.yaml")
	writeStatusInvoice(t, workingCopy, "CUST-001-001", "editing")
	source := testfixture.ReadFile(t, workingCopy) + "_invox:\n  archive_path: .history/customer-a/first.20261005T123045Z.yaml\n"
	if err := os.WriteFile(workingCopy, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := service(t, h, cmdutil.Files{}, filepath.Dir(workingCopy), time.Now()).Archive(workingCopy, billing.ArchiveOptions{Replace: true})
	if err == nil || !strings.Contains(err.Error(), "is a backup in .history, not an archived invoice") {
		t.Fatalf("ArchiveInvoice error = %v, want backup refusal", err)
	}
	if _, err := os.Stat(workingCopy); err != nil {
		t.Fatalf("working copy should stay in place: %v", err)
	}
}

// writePDF compiles a .tex file by writing a PDF next to it, as tectonic
// does.
type writePDF struct{}

func (writePDF) Compile(_ context.Context, sourcePath string) (string, error) {
	pdf := strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".pdf"
	return pdf, os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644)
}
