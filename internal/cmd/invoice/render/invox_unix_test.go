//go:build !windows

package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestRenderWritesPublicTex(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	outputPath := filepath.Join(t.TempDir(), "out", "invoice.tex")
	umask := testfixture.ProcessUmask(t)

	exitCode, stdout, stderr := x.Run([]string{
		"render", "-i", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Rendered " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	testfixture.AssertFileMode(t, outputPath, 0o644&^umask)
	testfixture.AssertFileMode(t, filepath.Dir(outputPath), 0o755&^umask)
}

func TestRenderCopiesNestedAssetDirsWithSourceMode(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	templateDir := filepath.Dir(fx.Template)
	fontsDir := filepath.Join(templateDir, "assets", "fonts")
	if err := os.MkdirAll(fontsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(templateDir, "fonts", "Ubuntu-Regular.ttf"), filepath.Join(fontsDir, "Ubuntu-Regular.ttf")); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Dir(fontsDir), fontsDir} {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	template, err := os.ReadFile(fx.Template)
	if err != nil {
		t.Fatal(err)
	}
	template = []byte(strings.Replace(string(template), "Path=fonts/", "Path=assets/fonts/", 1))
	if err := os.WriteFile(fx.Template, template, 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "invoice.tex")
	umask := testfixture.ProcessUmask(t)

	exitCode, stdout, stderr := x.Run([]string{
		"render", "-i", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Rendered " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	testfixture.AssertFileMode(t, filepath.Join(outputDir, "assets"), 0o700&^umask)
	testfixture.AssertFileMode(t, filepath.Join(outputDir, "assets", "fonts"), 0o700&^umask)
	testfixture.AssertFileMode(t, filepath.Join(outputDir, "assets", "fonts", "Ubuntu-Regular.ttf"), 0o644&^umask)
}
