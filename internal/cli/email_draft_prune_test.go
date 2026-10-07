package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestPruneEmailDraftsRemovesOnlyStaleDraftDirectories(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-25 * time.Hour)
	outside := t.TempDir()

	mkdir := func(name string, mtime time.Time) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "invoice.eml"), []byte("draft"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	mkdir("invox-email-111", stale)
	mkdir("invox-email-222", now.Add(-time.Hour))
	mkdir("invox-build-333", stale)
	if err := os.WriteFile(filepath.Join(dir, "invox-email-file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "invox-email-file"), stale, stale); err != nil {
		t.Fatal(err)
	}
	want := []string{"invox-build-333", "invox-email-222", "invox-email-file"}
	if err := os.Symlink(outside, filepath.Join(dir, "invox-email-link")); err == nil {
		want = append(want, "invox-email-link")
	}

	pruneEmailDrafts(dir, now.Add(-emailDraftMaxAge))
	pruneEmailDrafts(dir, now.Add(-emailDraftMaxAge))

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("left %q, want %q", got, want)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("symlink target was touched: %v", err)
	}
}

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
