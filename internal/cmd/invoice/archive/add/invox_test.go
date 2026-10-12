package add_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestArchiveReplaceWithoutTerminalRequiresYes(t *testing.T) {
	x := clitest.New(t)

	e := x.EditArchive()
	x.Stdin(false, "y\n")

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := "error: archiving first.yaml replaces archived invoice " + e.ArchivedPath + "; pass --yes to replace it (stdin is not a terminal)\n" +
		"Run 'invox archive add --help' for usage.\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	e.AssertUnchanged(t)
}

func TestArchiveReplaceOnTerminalDeclined(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "no", input: "n\n"},
		{name: "empty answer", input: "\n"},
		{name: "end of input", input: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			e := x.EditArchive()
			x.Stdin(true, tc.input)

			exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml"})
			if exitCode != 2 {
				t.Fatalf("exitCode = %d, want 2, stderr=%q", exitCode, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			prompt := "Replace archived invoice " + e.ArchivedPath + "? The previous version is kept in " +
				filepath.Join(e.ArchiveDir, ".history") + ". [y/N] "
			if !strings.HasPrefix(stderr, prompt) {
				t.Fatalf("stderr = %q, want it to start with prompt %q", stderr, prompt)
			}
			if !strings.HasSuffix(stderr, "not archived; the archive was not changed; pass --yes to replace without asking\n") {
				t.Fatalf("stderr = %q, want abort notice", stderr)
			}
			e.AssertUnchanged(t)
		})
	}
}

func TestArchiveReplaceOnTerminalConfirmed(t *testing.T) {
	x := clitest.New(t)

	e := x.EditArchive()
	x.Stdin(true, "y\n")

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := e.ArchivedPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	backupPath := e.AssertReplaced(t)
	prompt := "Replace archived invoice " + e.ArchivedPath + "? The previous version is kept in " +
		filepath.Join(e.ArchiveDir, ".history") + ". [y/N] "
	if want := prompt + e.ReplacedNotice(backupPath) + "Archived first.yaml -> " + e.ArchivedPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestArchiveReplaceWithYesKeepsBackup(t *testing.T) {
	x := clitest.New(t)

	e := x.EditArchive()
	// --yes answers the question, so the declining input is never read.
	x.Stdin(true, "n\n")

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := e.ArchivedPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	backupPath := e.AssertReplaced(t)
	if want := e.ReplacedNotice(backupPath) + "Archived first.yaml -> " + e.ArchivedPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}

	exitCode, stdout, stderr = x.Run([]string{"archive", "list"})
	if exitCode != 0 {
		t.Fatalf("archive list: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "first.yaml\tCUST-001\t2026-03-06\tarchived\n"; stdout != want {
		t.Fatalf("archive list: stdout = %q, want %q (backups must not be listed)", stdout, want)
	}
}

func TestArchiveReplaceWithYesStillValidates(t *testing.T) {
	x := clitest.New(t)

	e := x.EditArchive()
	testfixture.WriteNumberedInvoice(t, e.ArchiveDir, "second.yaml", "CUST-001-002", "archived")
	renumbered := strings.Replace(testfixture.ReadFile(t, e.WorkingCopy), "number: CUST-001-001", "number: CUST-001-002", 1)
	if err := os.WriteFile(e.WorkingCopy, []byte(renumbered), 0o644); err != nil {
		t.Fatalf("WriteFile(working copy) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "invoice number CUST-001-002 is already used by archived invoice "+filepath.Join(e.ArchiveDir, "second.yaml")) {
		t.Fatalf("stderr = %q, want duplicate-number error", stderr)
	}
	e.AssertUnchanged(t)
}

func TestArchiveAddHelpShowsShortFlags(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml or -i, --input PATH",
		"invox archive add [INVOICE.yaml] [flags]",
		"$ invox archive add invoice.yaml",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveRequiresPositionalOrFlagInput(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"archive", "add"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "error: missing required input: INVOICE.yaml or -i, --input\nRun 'invox archive add --help' for usage.\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestArchiveMovesBuiltInvoiceToArchiveDir(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: built
  period: Leistungszeitraum
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"archive",
		"add",
		invoicePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	archivePath := filepath.Join(archiveDir, filepath.Base(invoicePath))
	if want := "Archived " + invoicePath + " -> " + archivePath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != archivePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, archivePath+"\n")
	}
	if _, err := os.Stat(invoicePath); err == nil {
		t.Fatalf("source invoice should have been removed: %s", invoicePath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(invoicePath) returned unexpected error: %v", err)
	}

	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if !strings.Contains(string(archivedSource), "status: archived") {
		t.Fatalf("archived invoice does not contain archived status:\n%s", string(archivedSource))
	}
}

func TestArchiveReplacesEditedArchivedInvoice(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	archivedPath := filepath.Join(archiveDir, "2026-03-06.yaml")
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
    description: Original archive
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
		"2026-03-06.yaml",
	})
	if exitCode != 0 {
		t.Fatalf("edit exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Editing " + archivedPath + " -> 2026-03-06.yaml\n"; stderr != want {
		t.Fatalf("edit stderr = %q, want %q", stderr, want)
	}
	if stdout != "2026-03-06.yaml\n" {
		t.Fatalf("edit stdout = %q, want %q", stdout, "2026-03-06.yaml\n")
	}

	editedPath := filepath.Join(workDir, "2026-03-06.yaml")
	editedSource, err := os.ReadFile(editedPath)
	if err != nil {
		t.Fatalf("ReadFile(editedPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(editedSource), "Original archive", "Updated archive", 1)
	if err := os.WriteFile(editedPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(editedPath) returned error: %v", err)
	}

	exitCode, stdout, stderr = x.Run([]string{
		"archive",
		"add",
		editedPath,
		"--yes",
	})
	if exitCode != 0 {
		t.Fatalf("archive exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.HasPrefix(stderr, "Replaced archived invoice "+archivedPath+"; previous version kept at ") {
		t.Fatalf("archive stderr = %q, want replacement notice", stderr)
	}
	if !strings.HasSuffix(stderr, "\nArchived 2026-03-06.yaml -> "+archivedPath+"\n") {
		t.Fatalf("archive stderr %q does not end with the re-archive summary", stderr)
	}
	if stdout != archivedPath+"\n" {
		t.Fatalf("archive stdout = %q, want %q", stdout, archivedPath+"\n")
	}
	if _, err := os.Stat(editedPath); err == nil {
		t.Fatalf("edited working copy should have been removed: %s", editedPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(editedPath) returned unexpected error: %v", err)
	}

	archivedSource, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("ReadFile(archivedPath) returned error: %v", err)
	}
	archivedText := string(archivedSource)
	for _, want := range []string{
		"status: archived",
		"Updated archive",
	} {
		if !strings.Contains(archivedText, want) {
			t.Fatalf("archived invoice does not contain %q:\n%s", want, archivedText)
		}
	}
	for _, forbidden := range []string{
		"status: editing",
		"_invox:",
	} {
		if strings.Contains(archivedText, forbidden) {
			t.Fatalf("archived invoice should not contain %q:\n%s", forbidden, archivedText)
		}
	}
}

func TestArchiveRejectsInvoiceWithoutBuiltStatus(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: draft
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"archive",
		"add",
		invoicePath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "invoice.status must be `built` before archiving") {
		t.Fatalf("stderr %q does not contain status validation", stderr)
	}
	if _, err := os.Stat(invoicePath); err != nil {
		t.Fatalf("source invoice should remain in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, filepath.Base(invoicePath))); err == nil {
		t.Fatal("invoice should not be moved into archive dir")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(archivePath) returned unexpected error: %v", err)
	}
}

func TestArchiveRefusesDuplicateInvoiceNumber(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
	archivedBefore := testfixture.ReadFile(t, archivedPath)

	workDir := t.TempDir()
	x.Chdir(workDir)
	testfixture.WriteNumberedInvoice(t, workDir, "second.yaml", "CUST-001-001", "built")

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "second.yaml"})
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
	if got := testfixture.ReadFile(t, archivedPath); got != archivedBefore {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(workDir, "second.yaml")); err != nil {
		t.Fatalf("refused invoice should stay in place: %v", err)
	}
}

func TestArchiveEditThenRearchiveKeepsSameNumber(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, _, stderr := x.Run([]string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", "first.yaml", "--yes"})
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

// TestArchiveDryRunNeedsNoConfirmation shows that --dry-run neither asks
// nor needs --yes, while the same run without it still does.
func TestArchiveDryRunNeedsNoConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		terminal bool
		args     []string
	}{
		{name: "terminal", terminal: true, args: []string{"archive", "add", "first.yaml", "-n"}},
		{name: "no terminal", terminal: false, args: []string{"archive", "add", "first.yaml", "-n"}},
		{name: "no input", terminal: true, args: []string{"archive", "add", "first.yaml", "-n", "--no-input"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			e := x.EditArchive()
			// A prompt would read this answer and replace the archive.
			x.Stdin(tc.terminal, "y\n")

			exitCode, stdout, stderr := x.Run(tc.args)
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if want := e.ArchivedPath + "\n"; stdout != want {
				t.Fatalf("stdout = %q, want %q", stdout, want)
			}
			if strings.Contains(stderr, "[y/N]") {
				t.Fatalf("stderr = %q, want no prompt", stderr)
			}
			e.AssertUnchanged(t)
		})
	}
}

// Archiving removes the `_invox` key whatever it holds, including a link
// that names no archived file.
func TestArchiveRemovesArchiveLinkThatNamesNoFile(t *testing.T) {
	want := `customer_id: CUST-003
invoice:
  number: CUST-003-001
  issue_date: 2026-03-07
  due_date: 2026-04-06
  status: archived
  period: March
  vat_percent: 20
positions:
  - name: Consulting
    description: Workshop
    unit_price: 500
    quantity: 1
`
	for name, link := range map[string]string{
		"empty mapping": "_invox: {}\n",
		"null":          "_invox:\n",
		"empty path":    "_invox: {archive_path: \"\"}\n",
	} {
		t.Run(name, func(t *testing.T) {
			x := clitest.New(t)

			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			workDir := t.TempDir()
			x.Chdir(workDir)
			if err := os.WriteFile(filepath.Join(workDir, "inv.yaml"), []byte(testfixture.Source("cust003-built.yaml")+link), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := x.Run([]string{"archive", "add", "inv.yaml"})
			archived := filepath.Join(archiveDir, "inv.yaml")
			if wantErr := "Archived inv.yaml -> " + archived + "\n"; exitCode != 0 || stdout != archived+"\n" || stderr != wantErr {
				t.Fatalf("archive add = exit %d, stdout %q, stderr %q; want exit 0, stdout %q, stderr %q", exitCode, stdout, stderr, archived+"\n", wantErr)
			}
			if got := testfixture.ReadFile(t, archived); got != want {
				t.Fatalf("archived invoice =\n%s\nwant\n%s", got, want)
			}
		})
	}
}
