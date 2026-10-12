//go:build !windows

package add_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestArchiveCreatesPrivateArchive(t *testing.T) {
	x := clitest.New(t)

	archiveDir := filepath.Join(t.TempDir(), "archive")
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  status: built\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testfixture.SetUmask(t, 0o077)

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
	testfixture.AssertFileMode(t, archiveDir, 0o700)
	testfixture.AssertFileMode(t, archivePath, 0o600)
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
	umask := testfixture.ProcessUmask(t)

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
	testfixture.AssertFileMode(t, filepath.Join(root, "missing"), 0o755&^umask)
	testfixture.AssertFileMode(t, archiveDir, 0o700)
	testfixture.AssertFileMode(t, archivePath, 0o600)
}
