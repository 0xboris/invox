package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestRenderAcceptsTemplateFilenameFromGlobalConfig(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	configPath := x.WriteConfig("")
	configDir := filepath.Dir(configPath)

	templatePath := filepath.Join(configDir, "multi_vat.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Invoice @@INVOICE_NUMBER@@
Customer @@CUSTOMER_NAME@@
@@VAT_SUMMARY_ROWS@@
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "template.tex"), []byte("starter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(template.tex) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	exitCode, stdout, stderr := x.Run([]string{
		"render",
		"-i", fx.Invoice,
		"-o", outputPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", "multi_vat.tex",
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

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	for _, want := range []string{
		"Invoice CUST-001-001",
		"Customer Appsters GmbH",
		"VAT (20\\%):",
	} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("rendered output %q does not contain %q", string(rendered), want)
		}
	}
}

func TestRenderRequiresInputOnly(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"render"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "error: missing required input: INVOICE.yaml or -i, --input\nRun 'invox render --help' for usage.\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestRenderHelpShowsShortFlags(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"render", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"-i, --input PATH",
		"-o, --output string",
		"-c, --customers string",
		"-u, --issuer string",
		"-t, --template string",
		"schema/docs: run `invox help customers`",
		"schema/docs: run `invox help issuer`",
		"invoice.tex in the current directory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestRenderDefaultsOutputToInvoiceTex(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"render",
		"-i", fx.Invoice,
		"-c", fx.Customers,
		"-u", fx.Issuer,
		"-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Rendered invoice.tex for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "invoice.tex\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "invoice.tex\n")
	}
	if _, err := os.Stat(filepath.Join(workDir, "invoice.tex")); err != nil {
		t.Fatalf("default invoice.tex was not created: %v", err)
	}
}
