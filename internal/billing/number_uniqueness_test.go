package billing_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

func TestArchiveInvoiceReturnsDuplicateInvoiceNumberError(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeArchivedInvoice(t, archiveDir, "first.yaml", "CUST-001-001")

	invoicePath := filepath.Join(t.TempDir(), "second.yaml")
	writeStatusInvoice(t, invoicePath, "CUST-001-001", "built")

	_, err := h.service(t, cmdutil.Files{}, filepath.Dir(invoicePath), time.Now()).Archive(invoicePath, billing.ArchiveOptions{})
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

	workDir := t.TempDir()
	opened, err := h.service(t, cmdutil.Files{}, workDir, time.Time{}).EditArchived("first.yaml", workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	workingCopy := opened.Path
	if err := setInvoiceNumber(workingCopy, "CUST-001-002"); err != nil {
		t.Fatalf("writeInvoiceNumber returned error: %v", err)
	}

	_, err = h.service(t, cmdutil.Files{}, filepath.Dir(workingCopy), time.Now()).Archive(workingCopy, billing.ArchiveOptions{})
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

// validate's duplicate check skips the archived file itself and the one a
// working copy was opened from, and finds the number anywhere else.
func TestCheckArchivedNumberUniqueIgnoresTheArchivedOriginal(t *testing.T) {
	t.Parallel()

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	files := cmdutil.Files{Customers: customersPath, Issuer: issuerPath}
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archived := filepath.Join(archiveDir, "first.yaml")
	if err := os.WriteFile(archived, []byte(readTestFile(t, invoicePath)), 0o644); err != nil {
		t.Fatal(err)
	}

	workDir := t.TempDir()
	opened, err := h.service(t, files, workDir, time.Time{}).EditArchived("first.yaml", workDir, billing.EditOptions{})
	if err != nil {
		t.Fatalf("EditArchived returned error: %v", err)
	}
	for _, path := range []string{opened.Path, archived} {
		result, err := h.service(t, files, filepath.Dir(path), time.Time{}).Validate(path)
		if err != nil || result.Duplicate != nil {
			t.Fatalf("Validate(%s) = duplicate %v, error %v; want neither", path, result.Duplicate, err)
		}
	}

	result, err := h.service(t, files, filepath.Dir(invoicePath), time.Time{}).Validate(invoicePath)
	var duplicate *invoice.DuplicateInvoiceNumberError
	if err != nil || !errors.As(result.Duplicate, &duplicate) || duplicate.ArchivedPath != archived {
		t.Fatalf("Validate(other file) = duplicate %v, error %v; want %s as the duplicate", result.Duplicate, err, archived)
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

	_, err := h.service(t, cmdutil.Files{}, filepath.Dir(invoicePath), time.Now()).Archive(invoicePath, billing.ArchiveOptions{})
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
