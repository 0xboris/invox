package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveRefusesDuplicateInvoiceNumber(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
	archivedBefore := readFileForTest(t, archivedPath)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	writeNumberedInvoice(t, workDir, "second.yaml", "CUST-001-001", "built")

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", "second.yaml"})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{
		"second.yaml: invoice number CUST-001-001 is already used by archived invoice " + archivedPath + "\n",
		"Run 'invox increment -i second.yaml' to give it the next free number, then archive it again.\n",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
		}
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("ReadDir(archiveDir) returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "first.yaml" {
		t.Fatalf("archive entries = %v, want only first.yaml", entries)
	}
	if got := readFileForTest(t, archivedPath); got != archivedBefore {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(workDir, "second.yaml")); err != nil {
		t.Fatalf("refused invoice should stay in place: %v", err)
	}
}

func TestBuildArchiveRefusesDuplicateInvoiceNumber(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	exitCode, stdout, stderr := captureRun(t, []string{
		"build", invoicePath, "--archive",
		"-c", customersPath, "-u", issuerPath, "-t", templatePath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{
		"is already used by archived invoice " + archivedPath,
		"Run 'invox increment -i ",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
		}
	}
	if _, err := os.Stat(filepath.Join(archiveDir, "invoice.yaml")); !os.IsNotExist(err) {
		t.Fatalf("duplicate invoice should not have been archived, Stat err = %v", err)
	}
}

func TestArchiveEditThenRearchiveKeepsSameNumber(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 0 {
		t.Fatalf("archive: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.HasPrefix(stderr, "Replaced archived invoice "+archivedPath+"; previous version kept at ") {
		t.Fatalf("archive: stderr = %q, want replacement notice", stderr)
	}
	if want := "\nArchived first.yaml -> " + archivedPath + "\n"; !strings.HasSuffix(stderr, want) {
		t.Fatalf("archive: stderr = %q, want it to end with %q", stderr, want)
	}
	if want := archivedPath + "\n"; stdout != want {
		t.Fatalf("archive: stdout = %q, want %q", stdout, want)
	}
}

func writeNumberedInvoice(t *testing.T, dir, name, invoiceNumber, status string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	source := strings.Join([]string{
		"customer_id: CUST-001",
		"invoice:",
		"  number: " + invoiceNumber,
		"  issue_date: \"2026-03-06\"",
		"  due_date: \"2026-04-05\"",
		"  status: " + status,
		"  vat_percent: 20",
		"  paid_amount: 0",
		"positions:",
		"  - name: Development",
		"    unit_price: 100",
		"    quantity: 1",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
	return path
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return string(source)
}
