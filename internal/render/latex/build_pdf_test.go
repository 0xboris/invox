package latex_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestBuildInvoicePDFCompilesTheRenderedTeXAndCopiesThePDF(t *testing.T) {
	fx := testfixture.WriteContext(t)
	inv, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
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

	if err := buildInvoicePDF(t, testfixture.NewHost(t), context.Background(), compile, fx.Template, outputPath, inv); err != nil {
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
	fx := testfixture.WriteContext(t)
	inv, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.pdf")
	compileErr := errors.New("tectonic exploded")

	err = buildInvoicePDF(t, testfixture.NewHost(t), context.Background(), func(context.Context, string) error { return compileErr }, fx.Template, outputPath, inv)

	if !errors.Is(err, compileErr) {
		t.Fatalf("BuildInvoicePDF error = %v, want %v", err, compileErr)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want no PDF", err)
	}
}

// buildInvoicePDF renders inv into the template at templatePath and builds
// it to outputPath, running compile on the .tex file as tectonic would.
func buildInvoicePDF(t *testing.T, h testfixture.Host, ctx context.Context, compile func(ctx context.Context, texPath string) error, templatePath, outputPath string, inv *invoice.Context) error {
	t.Helper()
	tmpl := template(t, h, templatePath)
	renderer := renderer(t, h)
	source, err := renderer.Render(tmpl, inv, billing.EPCFor(inv))
	if err != nil {
		return err
	}
	renderer.Compiler = compilerFunc(func(ctx context.Context, sourcePath string) (string, error) {
		if err := compile(ctx, sourcePath); err != nil {
			return "", err
		}
		return strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".pdf", nil
	})
	return renderer.Build(ctx, tmpl, source, outputPath)
}

type compilerFunc func(ctx context.Context, sourcePath string) (string, error)

func (f compilerFunc) Compile(ctx context.Context, sourcePath string) (string, error) {
	return f(ctx, sourcePath)
}
