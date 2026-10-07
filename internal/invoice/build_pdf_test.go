package invoice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInvoicePDFCompilesTheRenderedTeXAndCopiesThePDF(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath, _, _ := writeContextFixtures(t)
	inv, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "out", "2026-0021.pdf")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatal(err)
	}

	var texPath, texSource string
	compile := func(_ context.Context, path string) error {
		texPath = path
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		texSource = string(source)
		return os.WriteFile(strings.TrimSuffix(path, ".tex")+".pdf", []byte("%PDF-1.4 compiled\n"), 0o644)
	}

	if err := BuildInvoicePDF(context.Background(), compile, templatePath, outputPath, inv); err != nil {
		t.Fatalf("BuildInvoicePDF returned error: %v", err)
	}

	if filepath.Base(texPath) != "2026-0021.tex" || filepath.Dir(texPath) == filepath.Dir(outputPath) {
		t.Fatalf("compiled %q, want 2026-0021.tex in a temporary directory", texPath)
	}
	if !strings.Contains(texSource, "Invoice CUST-001-001") {
		t.Fatalf("compiled TeX does not contain the rendered invoice:\n%s", texSource)
	}
	pdf, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if string(pdf) != "%PDF-1.4 compiled\n" {
		t.Fatalf("output PDF = %q, want the compiled PDF", pdf)
	}
	if _, err := os.Stat(filepath.Dir(texPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(build directory) error = %v, want it removed", err)
	}
}

func TestBuildInvoicePDFReturnsTheCompileError(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath, _, _ := writeContextFixtures(t)
	inv, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.pdf")
	compileErr := errors.New("tectonic exploded")

	err = BuildInvoicePDF(context.Background(), func(context.Context, string) error { return compileErr }, templatePath, outputPath, inv)

	if !errors.Is(err, compileErr) {
		t.Fatalf("BuildInvoicePDF error = %v, want %v", err, compileErr)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want no PDF", err)
	}
}
