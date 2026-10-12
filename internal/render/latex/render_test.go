package latex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestRenderInvoiceMatchesExistingOutput(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t,
		fx.Customers,
		fx.Issuer,
		fx.Invoice)

	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, fx.Template, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(output) returned error: %v", err)
	}
	text := string(got)
	for _, want := range []string{
		"Invoice CUST-001-001",
		"Customer Appsters GmbH",
		"Terms Pay within 30 days",
		"Development",
		"Support",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
	if strings.Contains(text, "@@INVOICE_NUMBER@@") {
		t.Fatalf("rendered output still contains placeholder: %q", text)
	}
}

func TestRenderInvoiceRendersSplitCityAndPostalCodePlaceholders(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t,
		fx.Customers,
		fx.Issuer,
		fx.Invoice)

	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Issuer: @@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@
Customer: @@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@
`)+"\n"), 0o644); err != nil {
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
	text := string(rendered)
	for _, want := range []string{
		"Issuer: 1010 Vienna",
		"Customer: 1010 Vienna",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceRendersVATSummaryRowsAndPerLineVATRows(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Rows:
@@LINE_ITEMS_ROWS_WITH_VAT@@
Totals:
@@VAT_SUMMARY_ROWS@@
`)+"\n"), 0o644); err != nil {
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
	text := string(rendered)
	for _, want := range []string{
		"Development & Sprint work & 100,00 \\euro & 2 & 20\\% & 200,00 \\euro",
		"Support & QA & 10,00 \\euro & 1 & 10\\% & 10,00 \\euro",
		"VAT (10\\%): & 1,00 \\euro",
		"VAT (20\\%): & 40,00 \\euro",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceRendersCustomLineItemBlockWithoutDescriptionColumn(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Rows:
@@LINE_ITEMS_BEGIN@@
@@LINE_ITEM_NAME@@ & @@LINE_ITEM_UNIT_PRICE@@ & @@LINE_ITEM_VAT_RATE@@ & @@LINE_ITEM_LINE_TOTAL@@\\
@@LINE_ITEM_RULE@@
@@LINE_ITEMS_END@@
`)+"\n"), 0o644); err != nil {
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
	text := string(rendered)
	want := strings.TrimSpace(`
Rows:
Development & 100,00 \euro & 20\% & 200,00 \euro\\
\specialrule{0.2pt}{0pt}{0pt}
Support & 10,00 \euro & 10\% & 10,00 \euro\\
\specialrule{0.4pt}{0pt}{0pt}
`)
	if !strings.Contains(text, want) {
		t.Fatalf("rendered output %q does not contain %q", text, want)
	}
	for _, unwanted := range []string{"Sprint work", "QA"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered output %q unexpectedly contains %q", text, unwanted)
		}
	}
}

func TestRenderInvoiceCustomLineItemBlockDoesNotLeaveBlankLineBeforeFollowingContent(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Rows:
@@LINE_ITEMS_BEGIN@@
@@LINE_ITEM_NAME@@\\
@@LINE_ITEMS_END@@
After
`)+"\n"), 0o644); err != nil {
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
	text := string(rendered)
	if !strings.Contains(text, "Development\\\\\nSupport\\\\\nAfter") {
		t.Fatalf("rendered output %q does not keep following content directly after the repeated block", text)
	}
	if strings.Contains(text, "Support\\\\\n\nAfter") {
		t.Fatalf("rendered output %q leaves an extra blank line before following content", text)
	}
}

func TestRenderInvoiceInlineCustomLineItemBlockPreservesLeadingNewlineInBody(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("Header@@LINE_ITEMS_BEGIN@@\n@@LINE_ITEM_NAME@@@@LINE_ITEMS_END@@\nFooter\n"), 0o644); err != nil {
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
	text := string(rendered)
	if !strings.Contains(text, "Header\nDevelopment\nSupport\nFooter\n") {
		t.Fatalf("rendered output %q does not preserve the block body's leading newline in inline usage", text)
	}
	if strings.Contains(text, "HeaderDevelopment") {
		t.Fatalf("rendered output %q incorrectly concatenates inline block content onto the prefix", text)
	}
}

func TestRenderInvoiceUsesCustomVATLabelInSummaryRows(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Issuer)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  payment_terms_text: Pay within 30 days\n", "  payment_terms_text: Pay within 30 days\n  vat_label: VAT & GST\n", 1)
	if err := os.WriteFile(fx.Issuer, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Totals:
Label: @@VAT_LABEL@@
@@VAT_SUMMARY_ROWS@@
`)+"\n"), 0o644); err != nil {
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
	text := string(rendered)
	for _, want := range []string{
		"Label: VAT \\& GST",
		"VAT \\& GST (20\\%): & 42,00 \\euro",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceRejectsLegacyVATPlaceholders(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("VAT @@VAT_RATE@@ @@VAT_AMOUNT@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for legacy VAT placeholders")
	}
	if want := templatePath + ": @@VAT_RATE@@: unknown placeholder\n@@VAT_AMOUNT@@: unknown placeholder"; err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRenderInvoiceRejectsLegacyCityAndPostalCodePlaceholders(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@ @@CUSTOMER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for legacy city/postal placeholders")
	}
	if want := templatePath + ": @@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder\n@@CUSTOMER_CITY_AND_POSTAL_CODE@@: unknown placeholder"; err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRenderInvoiceRejectsUnmatchedLineItemBlockPlaceholders(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@LINE_ITEMS_BEGIN@@\n@@LINE_ITEM_NAME@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for unmatched line-item block placeholder")
	}
	want := "@@LINE_ITEMS_BEGIN@@: missing matching @@LINE_ITEMS_END@@"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

func TestRenderInvoiceRejectsLineItemPlaceholdersOutsideCustomBlock(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@LINE_ITEM_NAME@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for line-item placeholder outside custom block")
	}
	want := "@@LINE_ITEM_NAME@@: only supported inside @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
