package latex_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestRenderInvoiceKeepsLeadingStarAndBracketAfterLineBreak(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	ctx.Customer.Address = invoice.Address{Street: "*Hinterhof", City: "Vienna"}
	ctx.Header.Period = "[Q1] 2026"

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	template := "@@CUSTOMER_CITY@@\\\\\n@@CUSTOMER_STREET@@\\\\\nPeriod:\\\\ @@PERIOD_LABEL@@\n"
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}
	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	want := "Vienna\\\\\n{}*Hinterhof\\\\\nPeriod:\\\\ {}[Q1] 2026\n"
	if string(rendered) != want {
		t.Fatalf("rendered = %q, want %q", rendered, want)
	}
}
