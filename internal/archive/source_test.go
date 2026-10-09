package archive_test

import (
	"os"
	"path/filepath"
	"strings"
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

	source, _, err := h.service(t).Archives.Source(filepath.Join(t.TempDir(), "BL00210001.pdf"))
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if want := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml"); source != want {
		t.Fatalf("InvoicePath = %q, want %q", source, want)
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

	source, _, err := h.service(t).Archives.Source(filepath.Join(t.TempDir(), "BL00210001.pdf"))
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if source != archivedInvoicePath {
		t.Fatalf("InvoicePath = %q, want %q", source, archivedInvoicePath)
	}
}

func TestResolveEmailDraftPathsRejectsAmbiguousArchiveMatches(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	for _, path := range []string{
		filepath.Join(archiveDir, "customer-a", "BL00210001.yaml"),
		filepath.Join(archiveDir, "customer-b", "BL00210001.yaml"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(filepath.Dir(path)) returned error: %v", err)
		}
		if err := os.WriteFile(path, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	_, _, err := h.service(t).Archives.Source(filepath.Join(t.TempDir(), "BL00210001.pdf"))
	if err == nil {
		t.Fatal("ResolveEmailDraftPaths returned nil error for ambiguous archive matches")
	}
	if !strings.Contains(err.Error(), "multiple archived invoice YAML files match") {
		t.Fatalf("error %q does not contain ambiguity message", err.Error())
	}
}
