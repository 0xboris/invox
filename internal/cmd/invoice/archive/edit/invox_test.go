package edit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// An archived invoice with a key invox does not know opens with `archive
// edit`; validate then reports the key with its line and what to do, and
// passes once the key is removed.
func TestArchiveEditThenValidateReportsUnknownKey(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	testfixture.WriteContextInvoice(t, archivedPath, "archived")
	source := testfixture.ReadFile(t, archivedPath)
	if err := os.WriteFile(archivedPath, []byte("notes: edited\n"+source), 0o644); err != nil {
		t.Fatal(err)
	}

	workDir := t.TempDir()
	x.Chdir(workDir)
	exitCode, _, stderr := x.Run([]string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit exit code = %d, want 0, stderr=%q", exitCode, stderr)
	}

	workingCopy := filepath.Join(workDir, "first.yaml")
	edited := testfixture.ReadFile(t, workingCopy)
	before, _, found := strings.Cut(edited, "notes: edited")
	if !found {
		t.Fatalf("edited working copy has no notes key:\n%s", edited)
	}
	line := strings.Count(before, "\n") + 1
	validate := []string{"validate", "-i", "first.yaml", "-c", fx.Customers, "-u", fx.Issuer}

	exitCode, stdout, stderr := x.Run(validate)
	want := fmt.Sprintf("error: first.yaml:%d: unknown key \"notes\"\n"+
		"Remove the unknown keys or fix their spelling; 'invox help defaults' lists the supported fields.\n", line)
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("validate = exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", exitCode, stdout, stderr, want)
	}

	if err := os.WriteFile(workingCopy, []byte(strings.Replace(edited, "notes: edited\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	exitCode, _, stderr = x.Run(validate)
	if exitCode != 0 || !strings.HasPrefix(stderr, "Validation OK: CUST-001-001 for CUST-001") {
		t.Fatalf("validate without the key = exit %d, stderr %q; want exit 0 and Validation OK", exitCode, stderr)
	}
}

func TestArchiveEditHelpShowsUsage(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"archive", "edit", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox archive edit FILENAME",
		"archive.dir",
		"invoice.status set to editing",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveEditCopiesArchivedInvoiceToCurrentDir(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	archivedPath := filepath.Join(archiveDir, "customer-a", "2026-03-06.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive subdir) returned error: %v", err)
	}
	if err := os.WriteFile(archivedPath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: archived
  period: March 2026
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedPath) returned error: %v", err)
	}

	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"archive",
		"edit",
		"customer-a/2026-03-06.yaml",
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	editedPath := filepath.Join(workDir, "2026-03-06.yaml")
	if want := "Editing " + archivedPath + " -> 2026-03-06.yaml\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "2026-03-06.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "2026-03-06.yaml\n")
	}
	editedSource, err := os.ReadFile(editedPath)
	if err != nil {
		t.Fatalf("ReadFile(editedPath) returned error: %v", err)
	}
	editedText := string(editedSource)
	for _, want := range []string{
		"status: editing",
		"_invox:",
		"archive_path: customer-a/2026-03-06.yaml",
	} {
		if !strings.Contains(editedText, want) {
			t.Fatalf("edited invoice does not contain %q:\n%s", want, editedText)
		}
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived invoice should remain in place: %v", err)
	}
}

func TestArchiveEditForceReplacesWorkingCopy(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
	workDir := t.TempDir()
	workingCopy := testfixture.WriteNumberedInvoice(t, workDir, "first.yaml", "CUST-001-099", "editing")
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{"archive", "edit", "first.yaml", "--force"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "first.yaml\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if want := "Editing " + archivedPath + " -> first.yaml\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	got := testfixture.ReadFile(t, workingCopy)
	if !strings.Contains(got, "number: CUST-001-001") || !strings.Contains(got, "archive_path: first.yaml") {
		t.Fatalf("working copy was not replaced by the archived invoice:\n%s", got)
	}
}
