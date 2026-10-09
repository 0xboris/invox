//go:build !windows

package cli_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
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

func TestArchiveCreatesPrivateArchive(t *testing.T) {
	x := clitest.New(t)

	archiveDir := filepath.Join(t.TempDir(), "archive")
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  status: built\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	setUmask(t, 0o077)

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", invoicePath})
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
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	outputPath := filepath.Join(t.TempDir(), "out", "invoice.tex")
	umask := processUmask(t)

	exitCode, stdout, stderr := x.Run([]string{
		"render", "-i", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
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
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	templateDir := filepath.Dir(fx.Template)
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
	template, err := os.ReadFile(fx.Template)
	if err != nil {
		t.Fatal(err)
	}
	template = []byte(strings.Replace(string(template), "Path=fonts/", "Path=assets/fonts/", 1))
	if err := os.WriteFile(fx.Template, template, 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "invoice.tex")
	umask := processUmask(t)

	exitCode, stdout, stderr := x.Run([]string{
		"render", "-i", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
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
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	x.ExpectOpener(nil)
	outputPath := filepath.Join(t.TempDir(), "draft.eml")

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer,
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

func TestArchiveCreatesMissingArchiveParentsPublic(t *testing.T) {
	x := clitest.New(t)

	root := t.TempDir()
	archiveDir := filepath.Join(root, "missing", "archive")
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  status: built\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	umask := processUmask(t)

	exitCode, stdout, stderr := x.Run([]string{"archive", "add", invoicePath})
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
