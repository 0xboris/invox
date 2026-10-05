package invoice

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveInvoiceReturnsDuplicateInvoiceNumberError(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeArchivedInvoiceMarkdown(t, archiveDir, "first.md", "CUST-001-001")

	invoicePath := filepath.Join(t.TempDir(), "second.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-001", "built")

	_, err := ArchiveInvoice(invoicePath)
	var duplicate *DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	want := DuplicateInvoiceNumberError{InvoicePath: invoicePath, InvoiceNumber: "CUST-001-001", ArchivedPath: archivedPath}
	if *duplicate != want {
		t.Fatalf("duplicate = %+v, want %+v", *duplicate, want)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, "second.yaml")); !os.IsNotExist(err) {
		t.Fatalf("duplicate invoice should not have been archived, Stat err = %v", err)
	}
	if _, err := os.Stat(invoicePath); err != nil {
		t.Fatalf("refused invoice should stay in place: %v", err)
	}
}

func TestArchiveInvoiceRefusesEditedCopyRenumberedToAnotherArchivedInvoice(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")
	secondPath := filepath.Join(archiveDir, "second.yaml")
	writeStatusInvoice(t, secondPath, "CUST-001-002", "archived")

	workingCopy, _, err := EditArchivedInvoice("first.yaml", t.TempDir())
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if err := writeInvoiceNumber(workingCopy, "CUST-001-002"); err != nil {
		t.Fatalf("writeInvoiceNumber returned error: %v", err)
	}

	_, err = ArchiveInvoice(workingCopy)
	var duplicate *DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	if duplicate.ArchivedPath != secondPath {
		t.Fatalf("ArchivedPath = %q, want %q", duplicate.ArchivedPath, secondPath)
	}
	source, err := os.ReadFile(filepath.Join(archiveDir, "first.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(first.yaml) returned error: %v", err)
	}
	if !strings.Contains(string(source), "number: CUST-001-001") {
		t.Fatalf("first.yaml changed:\n%s", source)
	}
}

func TestCheckArchivedNumberUniqueIgnoresTheArchivedOriginal(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")

	workingCopy, _, err := EditArchivedInvoice("first.yaml", t.TempDir())
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if err := CheckArchivedNumberUnique(workingCopy); err != nil {
		t.Fatalf("CheckArchivedNumberUnique(working copy) = %v, want nil", err)
	}
	if err := CheckArchivedNumberUnique(filepath.Join(archiveDir, "first.yaml")); err != nil {
		t.Fatalf("CheckArchivedNumberUnique(archived file) = %v, want nil", err)
	}
}

func writeStatusInvoice(t *testing.T, path, invoiceNumber, status string) {
	t.Helper()

	source := strings.Join([]string{
		"customer_id: CUST-001",
		"invoice:",
		"  number: " + invoiceNumber,
		"  issue_date: \"2026-03-06\"",
		"  status: " + status,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}
