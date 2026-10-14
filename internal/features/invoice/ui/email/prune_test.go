package email

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

	pruneEmailDrafts(dir, now.Add(-draftMaxAge))
	pruneEmailDrafts(dir, now.Add(-draftMaxAge))

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
