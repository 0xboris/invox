package build_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestBuildKeepsArchivedStatus(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()
	testfixture.WriteContextInvoice(t, fx.Invoice, "archived")
	original := testfixture.ReadFile(t, fx.Invoice)

	exitCode, stdout, stderr := x.Run([]string{
		"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf"
	if want := "Built " + pdfPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if want := pdfPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if got := testfixture.ReadFile(t, fx.Invoice); got != original {
		t.Fatalf("build changed the archived invoice:\n%s", got)
	}
}

func TestBuildArchiveReplacingArchivedInvoiceNeedsYes(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()
	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	testfixture.WriteContextInvoice(t, archivedPath, "archived")
	original := testfixture.ReadFile(t, archivedPath)

	workDir := t.TempDir()
	x.Chdir(workDir)
	exitCode, _, stderr := x.Run([]string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	buildArgs := []string{"build", "first.yaml", "--archive", "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template}

	exitCode, stdout, stderr := x.Run(buildArgs)
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "error: built first.pdf but archiving first.yaml replaces archived invoice " + archivedPath + "; pass --yes to replace it"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
	if got := testfixture.ReadFile(t, archivedPath); got != original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}

	x.ExpectTectonic()
	exitCode, stdout, stderr = x.Run(append(buildArgs, "--yes"))
	if exitCode != 0 {
		t.Fatalf("--yes: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "first.pdf\n"; stdout != want {
		t.Fatalf("--yes: stdout = %q, want %q", stdout, want)
	}
	backups, err := filepath.Glob(filepath.Join(archiveDir, ".history", "first.*.yaml"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %q (err %v), want exactly one", backups, err)
	}
	if want := "Replaced archived invoice " + archivedPath + "; previous version kept at " + backups[0] + "\n" +
		"Built first.pdf for CUST-001 (CUST-001-001)\nArchived first.yaml -> " + archivedPath + "\n"; stderr != want {
		t.Fatalf("--yes: stderr = %q, want %q", stderr, want)
	}
	if got := testfixture.ReadFile(t, backups[0]); got != original {
		t.Fatalf("backup = %q, want %q", got, original)
	}
}

// When build --archive would replace an archived invoice and the user can't
// or won't confirm, the error says once that the PDF was built.
func TestBuildArchiveReplacePromptStderr(t *testing.T) {
	for _, tc := range []struct {
		name       string
		terminal   bool
		answer     string
		wantStderr func(archivedPath, historyDir string) string
	}{
		{
			name: "no terminal",
			wantStderr: func(archivedPath, _ string) string {
				return "error: built first.pdf but archiving first.yaml replaces archived invoice " + archivedPath +
					"; pass --yes to replace it (stdin is not a terminal)\nRun 'invox build --help' for usage.\n"
			},
		},
		{
			// Empty input: tectonic runs on the same stdin before the prompt,
			// so an answer here would go to whichever reads it first.
			name: "declined at end of input", terminal: true, answer: "",
			wantStderr: func(archivedPath, historyDir string) string {
				return "Replace archived invoice " + archivedPath + "? The previous version is kept in " + historyDir + ". [y/N] \n" +
					"built first.pdf but not archived; the archive was not changed; pass --yes to replace without asking\n"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			fx := testfixture.WriteContext(t)
			x.ExpectTectonic()
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			archivedPath := filepath.Join(archiveDir, "first.yaml")
			testfixture.WriteContextInvoice(t, archivedPath, "archived")
			x.Chdir(t.TempDir())
			if exitCode, _, stderr := x.Run([]string{"archive", "edit", "first.yaml"}); exitCode != 0 {
				t.Fatalf("archive edit: exit code %d, stderr %q", exitCode, stderr)
			}

			x.Stdin(tc.terminal, tc.answer)
			exitCode, stdout, stderr := x.Run([]string{"build", "first.yaml", "--archive", "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template})

			if exitCode != 2 || stdout != "" {
				t.Fatalf("exit code %d, stdout %q; want 2, empty", exitCode, stdout)
			}
			if want := tc.wantStderr(archivedPath, filepath.Join(archiveDir, ".history")); stderr != want {
				t.Fatalf("stderr = %q\nwant     %q", stderr, want)
			}
		})
	}
}

func TestBuildKeepsTectonicOutputOffStdout(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	// tectonic prints its progress to stdout.
	x.Stub.Register("tectonic", func(cmd run.Cmd) error {
		fmt.Fprint(cmd.Stdout, "note: running TeX ...\nnote: writing `invoice.pdf`\n")
		testfixture.WriteFile(t, filepath.Join(cmd.Dir, "invoice.pdf"), "")
		return nil
	})

	exitCode, stdout, stderr := x.Run([]string{
		"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf"
	if want := pdfPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	want := "note: running TeX ...\nnote: writing `invoice.pdf`\nBuilt " + pdfPath + " for CUST-001 (CUST-001-001)\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestBuildHelpShowsInputBasedDefaultOutput(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"build", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml or -i, --input PATH",
		"-o, --output string",
		"--archive",
		"-c, --customers string",
		"-u, --issuer string",
		"-t, --template string",
		"the input path with .pdf extension",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestBuildRequiresPositionalOrFlagInput(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"build"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required input: INVOICE.yaml or -i, --input") {
		t.Fatalf("stderr %q does not contain missing input message", stderr)
	}
}

func TestBuildRejectsNonPDFOutput(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	exitCode, stdout, stderr := x.Run([]string{
		"build",
		"-i", fx.Invoice,
		"-o", filepath.Join(t.TempDir(), "invoice.tex"),
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "-o, --output must end with .pdf") {
		t.Fatalf("stderr %q does not contain output extension error", stderr)
	}
}

func TestBuildDefaultsPDFPathFromInputFile(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	workDir := t.TempDir()
	x.ExpectTectonic()

	x.Chdir(workDir)

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"build",
		customInvoicePath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	if want := "Built " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("default output PDF was not created: %v", err)
	}
	updatedInvoice, err := os.ReadFile(customInvoicePath)
	if err != nil {
		t.Fatalf("ReadFile(customInvoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(updatedInvoice), "status: built") {
		t.Fatalf("invoice file was not marked built:\n%s", string(updatedInvoice))
	}
	if _, err := os.Stat(filepath.Join(workDir, "invoice.pdf")); err == nil {
		t.Fatal("build should not write invoice.pdf into the current working directory")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(workDir/invoice.pdf) returned unexpected error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(inputDir, "BL00210001.tex"),
		filepath.Join(inputDir, "logo.png"),
		filepath.Join(inputDir, "fonts"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("unexpected build artifact left behind: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s) returned unexpected error: %v", path, err)
		}
	}
}

func TestBuildWithArchiveMovesInvoiceToArchiveDir(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210002.yaml")
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"build",
		customInvoicePath,
		"--archive",
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	archivePath := filepath.Join(archiveDir, filepath.Base(customInvoicePath))
	if want := "Built " + outputPath + " for CUST-001 (CUST-001-001)\nArchived " + customInvoicePath + " -> " + archivePath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output PDF was not created: %v", err)
	}
	if _, err := os.Stat(customInvoicePath); err == nil {
		t.Fatalf("source invoice should have been archived: %s", customInvoicePath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(customInvoicePath) returned unexpected error: %v", err)
	}
	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if !strings.Contains(string(archivedSource), "status: archived") {
		t.Fatalf("archived invoice does not contain archived status:\n%s", string(archivedSource))
	}
}

func TestBuildDoesNotMarkInvoiceBuiltWhenPDFBuildFails(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonicFailure(1)

	sourceBefore, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"build",
		"-i", fx.Invoice,
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr == "" {
		t.Fatal("stderr should contain build failure output")
	}

	sourceAfter, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if string(sourceAfter) != string(sourceBefore) {
		t.Fatalf("failed build should not modify invoice file:\nbefore:\n%s\nafter:\n%s", string(sourceBefore), string(sourceAfter))
	}
	if strings.Contains(string(sourceAfter), "status: built") {
		t.Fatalf("failed build should not mark invoice built:\n%s", string(sourceAfter))
	}
}

func TestBuildWithArchiveLeavesInvoiceBuiltWhenArchiveFails(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210003.yaml")
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	archivePath := filepath.Join(archiveDir, filepath.Base(customInvoicePath))
	if err := os.WriteFile(archivePath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"build",
		customInvoicePath,
		"--archive",
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "built "+customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))]+".pdf but failed to archive "+customInvoicePath) {
		t.Fatalf("stderr %q does not contain archive failure context", stderr)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output PDF should still exist: %v", err)
	}
	sourceAfter, err := os.ReadFile(customInvoicePath)
	if err != nil {
		t.Fatalf("ReadFile(customInvoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(sourceAfter), "status: built") {
		t.Fatalf("invoice should remain built after archive failure:\n%s", string(sourceAfter))
	}
	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if string(archivedSource) != "existing" {
		t.Fatalf("existing archive file should remain unchanged, got %q", string(archivedSource))
	}
}

func TestBuildArchiveRefusesDuplicateInvoiceNumber(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	exitCode, stdout, stderr := x.Run([]string{
		"build", fx.Invoice, "--archive",
		"-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
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

func TestBuildExitsOneWhenTectonicExitsTwo(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonicFailure(2)

	exitCode, stdout, stderr := x.Run([]string{"build", "-i", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template})

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if want := "fake tectonic: forced failure\nerror: tectonic exited with status 2\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}
