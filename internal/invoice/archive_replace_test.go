package invoice

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// archiveBackupTime is the time the replacement tests archive at, so the
// backup file names are known.
var archiveBackupTime = time.Date(2026, 10, 5, 12, 30, 45, 0, time.UTC)

// editArchivedForTest opens archiveName with `archive edit` and changes the
// working copy, so re-archiving it is a real replacement.
func editArchivedForTest(t *testing.T, archiveName string) string {
	t.Helper()

	workingCopy, _, err := EditArchivedInvoice(archiveName, t.TempDir())
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	// Set notes instead of appending it, so editing an invoice that was
	// already edited once still writes a single notes key.
	document, err := loadYAMLDocument(workingCopy)
	if err != nil {
		t.Fatalf("loadYAMLDocument returned error: %v", err)
	}
	root, err := documentRootMapping(document, workingCopy)
	if err != nil {
		t.Fatalf("documentRootMapping returned error: %v", err)
	}
	setMappingString(root, "notes", "edited")
	if err := writeYAMLDocument(workingCopy, document); err != nil {
		t.Fatalf("writeYAMLDocument(%s) returned error: %v", workingCopy, err)
	}
	return workingCopy
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return string(source)
}

func TestArchiveInvoiceRefusesToReplaceWithoutReplaceOption(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, archivedPath, "CUST-001-001", "archived")
	original := readTestFile(t, archivedPath)
	workingCopy := editArchivedForTest(t, "first.yaml")

	_, err := ArchiveInvoice(time.Now(), workingCopy, ArchiveOptions{})
	var replaceErr *ArchiveReplaceError
	if !errors.As(err, &replaceErr) {
		t.Fatalf("ArchiveInvoice error = %v, want *ArchiveReplaceError", err)
	}
	if len(replaceErr.Paths) != 1 || replaceErr.Paths[0] != archivedPath {
		t.Fatalf("Paths = %q, want [%q]", replaceErr.Paths, archivedPath)
	}
	if want := filepath.Join(archiveDir, ".history"); replaceErr.HistoryDir != want {
		t.Fatalf("HistoryDir = %q, want %q", replaceErr.HistoryDir, want)
	}
	if got := readTestFile(t, archivedPath); got != original {
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
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, archivedPath, "CUST-001-001", "archived")
	original := readTestFile(t, archivedPath)

	result, err := ArchiveInvoice(archiveBackupTime, editArchivedForTest(t, "first.yaml"), ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("ArchiveInvoice returned error: %v", err)
	}
	backupPath := filepath.Join(archiveDir, ".history", "first.20261005T123045Z.yaml")
	if want := []ArchiveBackup{{Path: archivedPath, BackupPath: backupPath}}; len(result.Replaced) != 1 || result.Replaced[0] != want[0] {
		t.Fatalf("Replaced = %+v, want %+v", result.Replaced, want)
	}
	if got := readTestFile(t, backupPath); got != original {
		t.Fatalf("backup = %q, want the previous version %q", got, original)
	}
	if !strings.Contains(readTestFile(t, archivedPath), "notes: edited") {
		t.Fatalf("archived invoice was not replaced:\n%s", readTestFile(t, archivedPath))
	}

	// A second replacement in the same second gets its own backup, and the
	// first backup does not count as a duplicate number.
	result, err = ArchiveInvoice(archiveBackupTime, editArchivedForTest(t, "first.yaml"), ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("second ArchiveInvoice returned error: %v", err)
	}
	if want := filepath.Join(archiveDir, ".history", "first.20261005T123045Z-2.yaml"); len(result.Replaced) != 1 || result.Replaced[0].BackupPath != want {
		t.Fatalf("second Replaced = %+v, want backup %s", result.Replaced, want)
	}

	summaries, err := ListArchivedInvoices()
	if err != nil {
		t.Fatalf("ListArchivedInvoices returned error: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Filename != "first.yaml" {
		t.Fatalf("ListArchivedInvoices = %+v, want only first.yaml", summaries)
	}
}

func TestArchiveInvoiceReplaceBacksUpMarkdownOriginalInSubdirectory(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	markdownPath := filepath.Join(archiveDir, "customer-a", "first.md")
	if err := os.MkdirAll(filepath.Dir(markdownPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	yamlSource := filepath.Join(t.TempDir(), "first.yaml")
	writeStatusInvoice(t, yamlSource, "CUST-001-001", "archived")
	original := "---\n" + readTestFile(t, yamlSource) + "---\n\n# Archived invoice\n"
	if err := os.WriteFile(markdownPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile(first.md) returned error: %v", err)
	}

	result, err := ArchiveInvoice(archiveBackupTime, editArchivedForTest(t, "customer-a/first.md"), ArchiveOptions{Replace: true})
	if err != nil {
		t.Fatalf("ArchiveInvoice returned error: %v", err)
	}
	backupPath := filepath.Join(archiveDir, ".history", "customer-a", "first.20261005T123045Z.md")
	if len(result.Replaced) != 1 || result.Replaced[0] != (ArchiveBackup{Path: markdownPath, BackupPath: backupPath}) {
		t.Fatalf("Replaced = %+v, want %s backed up to %s", result.Replaced, markdownPath, backupPath)
	}
	if got := readTestFile(t, backupPath); got != original {
		t.Fatalf("backup = %q, want %q", got, original)
	}
	if _, err := os.Stat(markdownPath); !os.IsNotExist(err) {
		t.Fatalf("markdown original should be replaced by YAML, Stat err = %v", err)
	}
}

func TestArchiveInvoiceReplaceStillChecksDuplicateNumbers(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	firstPath := filepath.Join(archiveDir, "first.yaml")
	writeStatusInvoice(t, firstPath, "CUST-001-001", "archived")
	writeStatusInvoice(t, filepath.Join(archiveDir, "second.yaml"), "CUST-001-002", "archived")
	original := readTestFile(t, firstPath)

	workingCopy := editArchivedForTest(t, "first.yaml")
	if err := writeInvoiceNumber(workingCopy, "CUST-001-002"); err != nil {
		t.Fatalf("writeInvoiceNumber returned error: %v", err)
	}

	_, err := ArchiveInvoice(time.Now(), workingCopy, ArchiveOptions{Replace: true})
	var duplicate *DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	if got := readTestFile(t, firstPath); got != original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}
}

func TestArchiveHistoryIsIgnoredByNumberingAndDuplicateCheck(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")
	historyDir := filepath.Join(archiveDir, ".history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(.history) returned error: %v", err)
	}
	writeStatusInvoice(t, filepath.Join(historyDir, "old.20261005T123045Z.yaml"), "CUST-001-009", "archived")

	invoiceNumber, _, err := NextInvoiceNumber("CUST-001", "2026-03-06", map[string]any{}, 0)
	if err != nil {
		t.Fatalf("NextInvoiceNumber returned error: %v", err)
	}
	if invoiceNumber != "CUST-001-002" {
		t.Fatalf("NextInvoiceNumber = %q, want CUST-001-002 (backups must not count)", invoiceNumber)
	}

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-009", "built")
	if err := CheckArchivedNumberUnique(invoicePath); err != nil {
		t.Fatalf("CheckArchivedNumberUnique = %v, want nil (backups must not count)", err)
	}

	summaries, err := ListArchivedInvoices()
	if err != nil {
		t.Fatalf("ListArchivedInvoices returned error: %v", err)
	}
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
			path := filepath.Join(t.TempDir(), "invoice.yaml")
			writeStatusInvoice(t, path, "CUST-001-001", tc.status)

			if err := MarkInvoiceBuilt(path); err != nil {
				t.Fatalf("MarkInvoiceBuilt returned error: %v", err)
			}
			if source := readTestFile(t, path); !strings.Contains(source, "status: "+tc.want+"\n") {
				t.Fatalf("status = %s, want %s:\n%s", tc.status, tc.want, source)
			}
		})
	}
}

func TestArchiveHistoryPathsAreRefused(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
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
			_, _, err := EditArchivedInvoice(name, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "is a backup in .history, not an archived invoice") {
				t.Fatalf("EditArchivedInvoice error = %v, want backup refusal", err)
			}
		})
	}

	// A working copy whose archive_path points into .history is refused too,
	// so a backup cannot be overwritten by re-archiving.
	workingCopy := filepath.Join(t.TempDir(), "first.yaml")
	writeStatusInvoice(t, workingCopy, "CUST-001-001", "editing")
	source := readTestFile(t, workingCopy) + "_invox:\n  archive_path: .history/customer-a/first.20261005T123045Z.yaml\n"
	if err := os.WriteFile(workingCopy, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err := ArchiveInvoice(time.Now(), workingCopy, ArchiveOptions{Replace: true})
	if err == nil || !strings.Contains(err.Error(), "is a backup in .history, not an archived invoice") {
		t.Fatalf("ArchiveInvoice error = %v, want backup refusal", err)
	}
	if _, err := os.Stat(workingCopy); err != nil {
		t.Fatalf("working copy should stay in place: %v", err)
	}
}
