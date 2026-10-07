//go:build !windows

package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// setUmask sets the process umask for the rest of the test. A restrictive
// umask shows that a mode comes from invox and not from the umask.
func setUmask(t *testing.T, umask int) {
	t.Helper()
	old := syscall.Umask(umask)
	t.Cleanup(func() { syscall.Umask(old) })
}

func assertFileMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}

func TestBuildKeepsPrivateInvoiceModeAndWritesPublicPDF(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)
	if err := os.Chmod(invoicePath, 0o600); err != nil {
		t.Fatal(err)
	}
	setUmask(t, 0o077)

	exitCode, stdout, stderr := captureRun(t, []string{
		"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(invoicePath, ".yaml") + ".pdf"
	if want := "Built " + pdfPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != pdfPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, pdfPath+"\n")
	}
	updated, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "status: built") {
		t.Fatalf("invoice was not marked built:\n%s", updated)
	}
	assertFileMode(t, invoicePath, 0o600)
	assertFileMode(t, pdfPath, 0o644)
}

func TestIncrementKeepsPrivateInvoiceMode(t *testing.T) {
	customersPath, _, _ := writeDraftFixtures(t)
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: 2026-03-06\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(t.TempDir())+"\n")

	exitCode, stdout, stderr := captureRun(t, []string{"increment", "-i", invoicePath, "-c", customersPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Incremented " + invoicePath + " for CUST-001: CUST-001-009 -> CUST-001-010\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != invoicePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, invoicePath+"\n")
	}
	assertFileMode(t, invoicePath, 0o600)
}

func TestInitCreatesPrivateConfigDirAndSecrets(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	setUmask(t, 0o022)

	exitCode, stdout, stderr := captureRun(t, []string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\ncreated customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	assertFileMode(t, configDir, 0o700)
	for name, want := range map[string]fs.FileMode{
		"issuer.yaml":           0o600,
		"customers.yaml":        0o600,
		"config.yaml":           0o644,
		"invoice_defaults.yaml": 0o644,
		"template.tex":          0o644,
	} {
		assertFileMode(t, filepath.Join(configDir, name), want)
	}
}

func TestArchiveCreatesPrivateArchive(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  status: built\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	setUmask(t, 0o022)

	exitCode, stdout, stderr := captureRun(t, []string{"archive", invoicePath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	archivePath := filepath.Join(archiveDir, "invoice.yaml")
	if want := "Archived " + invoicePath + " -> " + archivePath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != archivePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, archivePath+"\n")
	}
	assertFileMode(t, archiveDir, 0o700)
	assertFileMode(t, archivePath, 0o600)
}

func TestRenderWritesPublicTex(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	outputPath := filepath.Join(t.TempDir(), "out", "invoice.tex")
	setUmask(t, 0o077)

	exitCode, stdout, stderr := captureRun(t, []string{
		"render", "-i", invoicePath, "-o", outputPath, "-c", customersPath, "-u", issuerPath, "-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Rendered " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	assertFileMode(t, outputPath, 0o644)
	assertFileMode(t, filepath.Dir(outputPath), 0o755)
}

func TestEmailWritesPublicDraft(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	f, stub := testFactory(t)
	expectOpener(stub, nil)
	outputPath := filepath.Join(t.TempDir(), "draft.eml")
	setUmask(t, 0o077)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"email", invoicePath, "-o", outputPath, "-c", customersPath, "-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	assertFileMode(t, outputPath, 0o644)
}
