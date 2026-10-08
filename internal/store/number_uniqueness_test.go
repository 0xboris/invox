package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
)

func TestArchiveInvoiceReturnsDuplicateInvoiceNumberError(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeArchivedInvoiceMarkdown(t, archiveDir, "first.md", "CUST-001-001")

	invoicePath := filepath.Join(t.TempDir(), "second.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-001", "built")

	_, err := h.ArchiveInvoice(time.Now(), invoicePath, ArchiveOptions{})
	var duplicate *invoice.DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	want := invoice.DuplicateInvoiceNumberError{InvoicePath: invoicePath, InvoiceNumber: "CUST-001-001", ArchivedPath: archivedPath}
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
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")
	secondPath := filepath.Join(archiveDir, "second.yaml")
	writeStatusInvoice(t, secondPath, "CUST-001-002", "archived")

	workingCopy, _, err := h.EditArchivedInvoice("first.yaml", t.TempDir(), EditArchiveOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if err := writeInvoiceNumber(workingCopy, "CUST-001-002"); err != nil {
		t.Fatalf("writeInvoiceNumber returned error: %v", err)
	}

	_, err = h.ArchiveInvoice(time.Now(), workingCopy, ArchiveOptions{})
	var duplicate *invoice.DuplicateInvoiceNumberError
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
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeStatusInvoice(t, filepath.Join(archiveDir, "first.yaml"), "CUST-001-001", "archived")

	workingCopy, _, err := h.EditArchivedInvoice("first.yaml", t.TempDir(), EditArchiveOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if err := h.CheckArchivedNumberUnique(workingCopy); err != nil {
		t.Fatalf("CheckArchivedNumberUnique(working copy) = %v, want nil", err)
	}
	if err := h.CheckArchivedNumberUnique(filepath.Join(archiveDir, "first.yaml")); err != nil {
		t.Fatalf("CheckArchivedNumberUnique(archived file) = %v, want nil", err)
	}
}

func TestArchiveInvoiceChecksSymlinkedArchiveDir(t *testing.T) {
	t.Parallel()

	realArchiveDir := t.TempDir()
	writeStatusInvoice(t, filepath.Join(realArchiveDir, "first.yaml"), "CUST-001-001", "archived")
	archiveDir := filepath.Join(t.TempDir(), "archive-link")
	if err := os.Symlink(realArchiveDir, archiveDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	invoicePath := filepath.Join(t.TempDir(), "second.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-001", "built")

	_, err := h.ArchiveInvoice(time.Now(), invoicePath, ArchiveOptions{})
	var duplicate *invoice.DuplicateInvoiceNumberError
	if !errors.As(err, &duplicate) {
		t.Fatalf("ArchiveInvoice error = %v, want *DuplicateInvoiceNumberError", err)
	}
	if want := filepath.Join(archiveDir, "first.yaml"); duplicate.ArchivedPath != want {
		t.Fatalf("ArchivedPath = %q, want %q", duplicate.ArchivedPath, want)
	}
	if _, err := os.Stat(filepath.Join(realArchiveDir, "second.yaml")); !os.IsNotExist(err) {
		t.Fatalf("duplicate invoice should not have been archived, Stat err = %v", err)
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
