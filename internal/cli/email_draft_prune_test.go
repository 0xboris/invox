package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEmailRemovesStaleTemporaryDraftsBeforeWritingANewOne(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	tempDir := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, tempDir)
	}
	staleDir := filepath.Join(tempDir, "invox-email-1")
	if err := os.Mkdir(staleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(staleDir, stale, stale); err != nil {
		t.Fatal(err)
	}
	f, stub := testFactory(t)
	opened := expectOpener(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"email", invoicePath, "-c", customersPath, "-u", issuerPath})

	if exitCode != 0 || stdout != "" {
		t.Fatalf("exit code, stdout = %d, %q, want 0, empty; stderr=%q", exitCode, stdout, stderr)
	}
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(stale draft directory) error = %v, want it removed", err)
	}
	if filepath.Dir(filepath.Dir(*opened)) != tempDir {
		t.Fatalf("opened %q, want a new draft directory in %q", *opened, tempDir)
	}
	if _, err := os.Stat(*opened); err != nil {
		t.Fatalf("Stat(new draft) returned error: %v, want it kept for the mail app", err)
	}
}
