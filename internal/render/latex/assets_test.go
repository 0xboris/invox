package latex_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/render/latex"
)

func TestRenderInvoiceCopiesTemplateAssetsToOutputDir(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, templatePath, _, _ := writeContextFixtures(t)
	ctx, err := loadContext(t,
		customersPath,
		issuerPath,
		invoicePath,
	)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "invoice.tex")
	if err := h.renderInvoice(t, templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(outputDir, "logo.png"),
		filepath.Join(outputDir, "fonts", "Ubuntu-Regular.ttf"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected copied asset %q: %v", path, err)
		}
	}
}

func TestCopyTemplateAssetsFallsBackToGlobalConfig(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	templateDir := filepath.Join(t.TempDir(), "template")
	outputDir := filepath.Join(t.TempDir(), "output")

	if err := os.MkdirAll(filepath.Join(configDir, "fonts"), 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir/fonts) returned error: %v", err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(templateDir) returned error: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(outputDir) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	if err := os.WriteFile(filepath.Join(configDir, "logo.png"), []byte("logo"), 0o644); err != nil {
		t.Fatalf("WriteFile(global logo) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "fonts", "Ubuntu-Regular.ttf"), []byte("font"), 0o644); err != nil {
		t.Fatalf("WriteFile(global font) returned error: %v", err)
	}

	templatePath := filepath.Join(templateDir, "invoice_template.tex")
	outputPath := filepath.Join(outputDir, "invoice.tex")
	rendered := "\\setmainfont{Ubuntu}[Path=fonts/,UprightFont=Ubuntu-Regular.ttf]\n\\includegraphics{logo.png}\n"

	if err := (latex.Renderer{}).Write(h.template(t, templatePath), rendered, outputPath); err != nil {
		t.Fatalf("copyTemplateAssets returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(outputDir, "logo.png"),
		filepath.Join(outputDir, "fonts", "Ubuntu-Regular.ttf"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected copied fallback asset %q: %v", path, err)
		}
	}
}
