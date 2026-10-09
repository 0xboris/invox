package store

import (
	"os"
	"path/filepath"
	"testing"
)

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
