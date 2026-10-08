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

// setUmask sets the process umask for the rest of the test. invox reads the
// umask once at start, so this only steers modes it does not set itself.
func setUmask(t *testing.T, umask int) {
	t.Helper()
	old := syscall.Umask(umask)
	t.Cleanup(func() { syscall.Umask(old) })
}

// processUmask returns the umask the test process started with, which invox
// clears from the modes of files and directories that are not private.
func processUmask(t *testing.T) fs.FileMode {
	t.Helper()
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	return fs.FileMode(umask)
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
	assertFileMode(t, pdfPath, 0o644&^processUmask(t))
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
	umask := processUmask(t)

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
		"config.yaml":           0o644 &^ umask,
		"invoice_defaults.yaml": 0o644 &^ umask,
		"template.tex":          0o644 &^ umask,
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
	setUmask(t, 0o077)

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
	umask := processUmask(t)

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
	assertFileMode(t, outputPath, 0o644&^umask)
	assertFileMode(t, filepath.Dir(outputPath), 0o755&^umask)
}

func TestRenderCopiesNestedAssetDirsWithSourceMode(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	templateDir := filepath.Dir(templatePath)
	fontsDir := filepath.Join(templateDir, "assets", "fonts")
	if err := os.MkdirAll(fontsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(templateDir, "fonts", "Ubuntu-Regular.ttf"), filepath.Join(fontsDir, "Ubuntu-Regular.ttf")); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Dir(fontsDir), fontsDir} {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	template, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	template = []byte(strings.Replace(string(template), "Path=fonts/", "Path=assets/fonts/", 1))
	if err := os.WriteFile(templatePath, template, 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "invoice.tex")
	umask := processUmask(t)

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
	assertFileMode(t, filepath.Join(outputDir, "assets"), 0o700&^umask)
	assertFileMode(t, filepath.Join(outputDir, "assets", "fonts"), 0o700&^umask)
	assertFileMode(t, filepath.Join(outputDir, "assets", "fonts", "Ubuntu-Regular.ttf"), 0o644&^umask)
}

func TestEmailWritesPublicDraft(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	f, stub := testFactory(t)
	expectOpener(stub, nil)
	outputPath := filepath.Join(t.TempDir(), "draft.eml")

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
	assertFileMode(t, outputPath, 0o644&^processUmask(t))
}

func TestInitCreatesMissingConfigParentsPublic(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(root, "missing", "cfg")
	configDir := filepath.Join(configHome, "invox")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	umask := processUmask(t)

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
	assertFileMode(t, filepath.Join(root, "missing"), 0o755&^umask)
	assertFileMode(t, configHome, 0o755&^umask)
	assertFileMode(t, configDir, 0o700)
}

func TestArchiveCreatesMissingArchiveParentsPublic(t *testing.T) {
	root := t.TempDir()
	archiveDir := filepath.Join(root, "missing", "archive")
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  status: built\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	umask := processUmask(t)

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
	assertFileMode(t, filepath.Join(root, "missing"), 0o755&^umask)
	assertFileMode(t, archiveDir, 0o700)
	assertFileMode(t, archivePath, 0o600)
}
