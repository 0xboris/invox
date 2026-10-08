package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEmailDraftPathsFollowsSymlinkedArchiveDir(t *testing.T) {
	t.Parallel()

	realArchiveDir := t.TempDir()
	archiveDir := filepath.Join(t.TempDir(), "archive-link")
	if err := os.Symlink(realArchiveDir, archiveDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedInvoicePath := filepath.Join(realArchiveDir, "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedInvoicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(archivedInvoicePath)) returned error: %v", err)
	}
	if err := os.WriteFile(archivedInvoicePath, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedInvoicePath) returned error: %v", err)
	}

	paths, err := h.ResolveEmailDraftPaths(filepath.Join(t.TempDir(), "BL00210001.pdf"), "", "")
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if want := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml"); paths.InvoicePath != want {
		t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, want)
	}
}

func TestResolveEmailDraftPathsSkipsArchiveHistory(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedInvoicePath := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml")
	for _, path := range []string{
		archivedInvoicePath,
		filepath.Join(archiveDir, ".history", "BL00210001.yaml"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(filepath.Dir(path)) returned error: %v", err)
		}
		if err := os.WriteFile(path, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	paths, err := h.ResolveEmailDraftPaths(filepath.Join(t.TempDir(), "BL00210001.pdf"), "", "")
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if paths.InvoicePath != archivedInvoicePath {
		t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, archivedInvoicePath)
	}
}

func TestResolveEmailDraftPathsIgnoresAMatchOnlyInArchiveHistory(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	historyPath := filepath.Join(archiveDir, ".history", "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(historyPath)) returned error: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(historyPath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "BL00210001.pdf")
	paths, err := h.ResolveEmailDraftPaths(pdfPath, "", "")
	want := pdfPath + ": no matching invoice YAML found next to the PDF or in archive.dir"
	if err == nil || err.Error() != want {
		t.Fatalf("ResolveEmailDraftPaths = %q, %v; want error %q", paths.InvoicePath, err, want)
	}
}
