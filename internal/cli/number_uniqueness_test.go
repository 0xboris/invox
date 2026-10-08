package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTwiceWithoutArchivingAllocatesDifferentNumbers(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	workDir := t.TempDir()
	chdirForTest(t, workDir)

	for _, tc := range []struct {
		output     string
		wantStdout string
		wantStderr string
	}{
		{output: "a.yaml", wantStdout: "a.yaml\n", wantStderr: "Created a.yaml for CUST-001 (CUST-001-001)\n"},
		{output: "b.yaml", wantStdout: "b.yaml\n", wantStderr: "Created b.yaml for CUST-001 (CUST-001-002)\n"},
		{output: "", wantStdout: "CUST-001-003.yaml\n", wantStderr: "Created CUST-001-003.yaml for CUST-001 (CUST-001-003)\n"},
	} {
		args := []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath}
		if tc.output != "" {
			args = append(args, "-o", tc.output)
		}
		exitCode, stdout, stderr := captureRun(t, args)
		if exitCode != 0 {
			t.Fatalf("new -o %q: exitCode = %d, want 0, stderr=%q", tc.output, exitCode, stderr)
		}
		if stderr != tc.wantStderr {
			t.Fatalf("new -o %q: stderr = %q, want %q", tc.output, stderr, tc.wantStderr)
		}
		if stdout != tc.wantStdout {
			t.Fatalf("new -o %q: stdout = %q, want %q", tc.output, stdout, tc.wantStdout)
		}
	}
}

func TestNewSkipsNumbersOfDraftsInOutputDirectory(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	chdirForTest(t, t.TempDir())

	outputDir := t.TempDir()
	writeNumberedInvoice(t, outputDir, "built.yaml", "CUST-001-004", "built")
	writeNumberedInvoice(t, outputDir, "editing.yaml", "CUST-001-009", "editing")

	exitCode, stdout, stderr := captureRun(t, []string{
		"new", "CUST-001",
		"-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath,
		"-o", filepath.Join(outputDir, "next.yaml"),
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "(CUST-001-005)") {
		t.Fatalf("stderr = %q, want number CUST-001-005", stderr)
	}
	if want := filepath.Join(outputDir, "next.yaml") + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

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

func TestValidateWarnsWhenNumberIsAlreadyArchived(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	exitCode, stdout, stderr := captureRun(t, []string{"validate", "-i", invoicePath, "-c", customersPath, "-u", issuerPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := "warning: invoice number CUST-001-001 is already used by archived invoice " + archivedPath +
		"; run 'invox increment -i " + invoicePath + "' before archiving\n" +
		"Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestValidateDoesNotWarnForUniqueNumber(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeNumberedInvoice(t, archiveDir, "other.yaml", "CUST-001-002", "archived")

	exitCode, stdout, stderr := captureRun(t, []string{"validate", "-i", invoicePath, "-c", customersPath, "-u", issuerPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestForgedReplacePathDoesNotExemptDuplicateNumber(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	// archive_replace_path without archive_path is not an `archive edit`
	// working copy, so it must not hide the duplicate.
	source := readFileForTest(t, invoicePath) + "_invox:\n  archive_replace_path: first.yaml\n"
	if err := os.WriteFile(invoicePath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	exitCode, _, stderr := captureRun(t, []string{"validate", "-i", invoicePath, "-c", customersPath, "-u", issuerPath})
	if exitCode != 0 {
		t.Fatalf("validate: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "warning: invoice number CUST-001-001 is already used by archived invoice "+archivedPath) {
		t.Fatalf("validate: stderr = %q, want duplicate warning", stderr)
	}

	source = strings.Replace(source, "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}
	exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", invoicePath})
	if exitCode != 1 {
		t.Fatalf("archive: exitCode = %d, want 1, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "is already used by archived invoice "+archivedPath) {
		t.Fatalf("archive: stderr = %q, want duplicate error", stderr)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, "invoice.yaml")); !os.IsNotExist(err) {
		t.Fatalf("duplicate invoice should not have been archived, Stat err = %v", err)
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived original should be kept: %v", err)
	}
}

func TestNewIgnoresUnrelatedAndOversizedYAMLInWorkingDirectory(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	workDir := t.TempDir()
	chdirForTest(t, workDir)

	for name, source := range map[string]string{
		"broken.yaml":    "invoice: [unclosed\n",
		"customers.yaml": "CUST-001:\n  name: Appsters GmbH\n",
		"list.yml":       "- one\n- two\n",
		"other.yaml":     "customer_id: CUST-002\ninvoice:\n  number: CUST-002-007\n  status: draft\n",
	} {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(source), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}
	oversized := readFileForTest(t, writeNumberedInvoice(t, workDir, "huge.yaml", "CUST-001-009", "draft")) +
		"# " + strings.Repeat("x", 1<<20) + "\n"
	if err := os.WriteFile(filepath.Join(workDir, "huge.yaml"), []byte(oversized), 0o644); err != nil {
		t.Fatalf("WriteFile(huge.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-001.yaml for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if want := "CUST-001-001.yaml\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestArchiveEditMarkdownThenRearchiveReplacesOriginal(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	markdownPath := filepath.Join(archiveDir, "first.md")
	markdown := "---\n" + readFileForTest(t, writeNumberedInvoice(t, t.TempDir(), "first.yaml", "CUST-001-001", "archived")) + "---\n\n# Archived invoice\n"
	if err := os.WriteFile(markdownPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("WriteFile(first.md) returned error: %v", err)
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.md"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 0 {
		t.Fatalf("archive: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.HasPrefix(stderr, "Replaced archived invoice "+markdownPath+"; previous version kept at ") {
		t.Fatalf("archive: stderr = %q, want replacement notice", stderr)
	}
	yamlPath := filepath.Join(archiveDir, "first.yaml")
	if want := "\nArchived first.yaml -> " + yamlPath + "\n"; !strings.HasSuffix(stderr, want) {
		t.Fatalf("archive: stderr = %q, want it to end with %q", stderr, want)
	}
	if want := yamlPath + "\n"; stdout != want {
		t.Fatalf("archive: stdout = %q, want %q", stdout, want)
	}
	if _, err := os.Stat(markdownPath); !os.IsNotExist(err) {
		t.Fatalf("markdown original should have been replaced, Stat err = %v", err)
	}
	if !strings.Contains(readFileForTest(t, yamlPath), "number: CUST-001-001") {
		t.Fatalf("re-archived invoice lost its number:\n%s", readFileForTest(t, yamlPath))
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
