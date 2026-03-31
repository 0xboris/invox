package invoice

import (
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDiscoverBaseDirFallsBackToStartDirWithoutProjectMarkers(t *testing.T) {
	goDir := t.TempDir()

	if got := DiscoverBaseDir(goDir); got != goDir {
		t.Fatalf("DiscoverBaseDir(%q) = %q, want %q", goDir, got, goDir)
	}
}

func TestLoadContextWithCurrentInvoice(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(
		customersPath,
		issuerPath,
		invoicePath,
	)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	if ctx.CustomerID != "CUST-001" {
		t.Fatalf("CustomerID = %q, want %q", ctx.CustomerID, "CUST-001")
	}
	if ctx.InvoiceNumber != "CUST-001-001" {
		t.Fatalf("InvoiceNumber = %q, want %q", ctx.InvoiceNumber, "CUST-001-001")
	}
	if ctx.CustomerEmail != "office@appsters.example" {
		t.Fatalf("CustomerEmail = %q, want %q", ctx.CustomerEmail, "office@appsters.example")
	}
	if ctx.Currency != "EUR" {
		t.Fatalf("Currency = %q, want %q", ctx.Currency, "EUR")
	}
	if ctx.TotalCents != 25200 {
		t.Fatalf("TotalCents = %d, want %d", ctx.TotalCents, 25200)
	}
	if ctx.OutstandingCents != 25200 {
		t.Fatalf("OutstandingCents = %d, want %d", ctx.OutstandingCents, 25200)
	}
	if len(ctx.LineItems) != 2 {
		t.Fatalf("len(LineItems) = %d, want 2", len(ctx.LineItems))
	}
}

func TestLoadContextRejectsLegacyInvoiceAliases(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}

	mutated := strings.NewReplacer(
		"  period: Leistungszeitraum", "  period_label: Leistungszeitraum",
		"  vat_percent: 20", "  vat_rate_percent: 20",
		"positions:", "line_items:",
	).Replace(string(source))
	legacyPath := filepath.Join(t.TempDir(), "legacy-invoice.yaml")
	if err := os.WriteFile(legacyPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(legacyPath) returned error: %v", err)
	}

	_, err = LoadContext(
		customersPath,
		issuerPath,
		legacyPath,
	)
	if err == nil {
		t.Fatal("LoadContext returned nil error for legacy aliases")
	}
	for _, want := range []string{
		"invoice.period_label: unsupported key; use invoice.period",
		"invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		"line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestLoadContextSupportsPerPositionVATOverrides(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}

	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	if got := len(ctx.VATBreakdowns); got != 2 {
		t.Fatalf("len(VATBreakdowns) = %d, want 2", got)
	}
	if got := formatQuantity(ctx.LineItems[0].VATRatePercent); got != "20" {
		t.Fatalf("LineItems[0].VATRatePercent = %q, want %q", got, "20")
	}
	if got := formatQuantity(ctx.LineItems[1].VATRatePercent); got != "10" {
		t.Fatalf("LineItems[1].VATRatePercent = %q, want %q", got, "10")
	}
	if ctx.SubtotalCents != 21000 {
		t.Fatalf("SubtotalCents = %d, want %d", ctx.SubtotalCents, 21000)
	}
	if ctx.VATAmountCents != 4100 {
		t.Fatalf("VATAmountCents = %d, want %d", ctx.VATAmountCents, 4100)
	}
	if ctx.TotalCents != 25100 {
		t.Fatalf("TotalCents = %d, want %d", ctx.TotalCents, 25100)
	}
	if got := formatQuantity(ctx.VATBreakdowns[0].RatePercent); got != "10" {
		t.Fatalf("VATBreakdowns[0].RatePercent = %q, want %q", got, "10")
	}
	if ctx.VATBreakdowns[0].NetCents != 1000 || ctx.VATBreakdowns[0].VATAmountCents != 100 {
		t.Fatalf("VATBreakdowns[0] = %+v, want net=1000 vat=100", ctx.VATBreakdowns[0])
	}
	if got := formatQuantity(ctx.VATBreakdowns[1].RatePercent); got != "20" {
		t.Fatalf("VATBreakdowns[1].RatePercent = %q, want %q", got, "20")
	}
	if ctx.VATBreakdowns[1].NetCents != 20000 || ctx.VATBreakdowns[1].VATAmountCents != 4000 {
		t.Fatalf("VATBreakdowns[1] = %+v, want net=20000 vat=4000", ctx.VATBreakdowns[1])
	}
}

func TestLoadContextFallsBackToCustomerDefaultVATRate(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)

	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  status: active
  email: office@appsters.example
  email_greeting: Dear Jane Doe,
  contact_person: Jane Doe
  address:
    street: Hauptstrasse 1
    postal_code: 1010
    city: Vienna
    country: Austria
  tax:
    vat_tax_id: ATU12345678
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  vat_percent: 20\n", "", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	if got := len(ctx.VATBreakdowns); got != 1 {
		t.Fatalf("len(VATBreakdowns) = %d, want 1", got)
	}
	if got := formatQuantity(ctx.LineItems[0].VATRatePercent); got != "13" {
		t.Fatalf("LineItems[0].VATRatePercent = %q, want %q", got, "13")
	}
	if ctx.VATAmountCents != 2730 {
		t.Fatalf("VATAmountCents = %d, want %d", ctx.VATAmountCents, 2730)
	}
	if ctx.TotalCents != 23730 {
		t.Fatalf("TotalCents = %d, want %d", ctx.TotalCents, 23730)
	}
}

func TestLoadContextRejectsMissingInvoiceNumber(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	invoicePath = writeInvoiceWithoutNumber(t, invoicePath)

	_, err := LoadContext(
		customersPath,
		issuerPath,
		invoicePath,
	)
	if err == nil {
		t.Fatalf("LoadContext returned nil error for missing invoice.number")
	}
	if !strings.Contains(err.Error(), "invoice.number: missing value") {
		t.Fatalf("error %q does not contain missing number message", err.Error())
	}
}

func TestRenderInvoiceMatchesExistingOutput(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(
		customersPath,
		issuerPath,
		invoicePath,
	)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(
		customersPath,
		issuerPath,
		invoicePath,
	)
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
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
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
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
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
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
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
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("Header@@LINE_ITEMS_BEGIN@@\n@@LINE_ITEM_NAME@@@@LINE_ITEMS_END@@\nFooter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  payment_terms_text: Pay within 30 days\n", "  payment_terms_text: Pay within 30 days\n  vat_label: VAT & GST\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
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
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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

func TestRenderInvoiceRendersEPCQRCode(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\documentclass{article}
\usepackage{qrcode}
\begin{document}
@@EPC_QR_CODE@@
\end{document}
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`\edef\invoxqrcodepayload{BCD\noexpand\?002\noexpand\?1\noexpand\?SCT\noexpand\?BKAUATWW\noexpand\?Boris Consulting\noexpand\?AT611904300234573201\noexpand\?EUR252.00`,
		`\noexpand\?\noexpand\?CUST-001-001}`,
		`\qrcode{\invoxqrcodepayload}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestQRCodePayloadTeXSourceUsesQrcodeEscapesForReservedCharacters(t *testing.T) {
	payload := []byte("A\\B^C~D%E#F&G_H$I{J}K\n")

	got := qrcodePayloadTeXSource(payload)
	want := strings.Join([]string{
		"A",
		`\noexpand\\`,
		"B",
		`\noexpand\^`,
		"C",
		`\noexpand\~`,
		"D",
		`\noexpand\%`,
		"E",
		`\noexpand\#`,
		"F",
		`\noexpand\&`,
		"G",
		`\noexpand\_`,
		"H",
		`\noexpand\$`,
		"I",
		`\noexpand\{`,
		"J",
		`\noexpand\}`,
		"K",
		`\noexpand\?`,
	}, "")
	if got != want {
		t.Fatalf("qrcodePayloadTeXSource(%q) = %q, want %q", payload, got, want)
	}
}

func TestCompactEPCAccountIdentifierRemovesUnicodeWhitespace(t *testing.T) {
	got := compactEPCAccountIdentifier(" \tAT61\u00a01904 3002\t3457 3201\n")
	want := "AT611904300234573201"
	if got != want {
		t.Fatalf("compactEPCAccountIdentifier returned %q, want %q", got, want)
	}
}

func TestRenderInvoiceEscapesReservedQRCodeCharactersInDefaultReference(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  number: CUST-001-001\n", "  number: 'INV-\\^~{}'\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\documentclass{article}
\usepackage{qrcode}
\begin{document}
@@EPC_QR_CODE@@
\end{document}
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	want := `INV-\noexpand\\\noexpand\^\noexpand\~\noexpand\{\noexpand\}}`
	if !strings.Contains(text, want) {
		t.Fatalf("rendered output %q does not contain %q", text, want)
	}
}

func TestRenderInvoiceAllowsInlineEPCQRCodePlacement(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\documentclass{article}
\usepackage{qrcode}
\begin{document}
\fbox{@@EPC_QR_CODE@@}
\end{document}
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`\fbox{{%`,
		`\qrcode{\invoxqrcodepayload}}}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
	if strings.Contains(text, "}%}") {
		t.Fatalf("rendered output %q unexpectedly comments out the trailing template brace", text)
	}
	if strings.Contains(text, `\qrcode{\invoxqrcodepayload}`+"\n") {
		t.Fatalf("rendered output %q unexpectedly leaves inline QR content followed by a space-producing newline", text)
	}
}

func TestRenderInvoiceRendersEPCQRAvailableAndLabelWhenEligible(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\nBefore\n@@EPC_QR_AVAILABLE@@\n@@EPC_QR_LABEL@@\n@@EPC_QR_CODE@@\nAfter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		"\n1\n",
		`Pay via EPC-QR`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
	for _, unwanted := range []string{
		`\vspace{0.5cm}`,
		`Pay via EPC-QR\\`,
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered output %q unexpectedly contains layout-specific EPC label LaTeX %q", text, unwanted)
		}
	}
}

func TestRenderInvoiceUsesConfiguredEPCQRCodeLabel(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  payment_terms_text: Pay within 30 days\n", ""+
		"  payment_terms_text: Pay within 30 days\n"+
		"  epc_qr:\n"+
		"    label: Zahlung per QR-Code\n"+
		"    purpose: SUPP\n"+
		"    information: Scan to pay this invoice\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_AVAILABLE@@\n@@EPC_QR_LABEL@@\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		"\n1\n",
		`Zahlung per QR-Code`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceAcceptsUnicodeWhitespaceInEPCAccountIdentifiers(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.ReplaceAll(string(source), "  iban: AT611904300234573201\n", "  iban: \"AT61\u00a01904\t3002 3457 3201\"\n")
	mutated = strings.ReplaceAll(mutated, "  bic: BKAUATWW\n", "  bic: \"BKAU\u00a0AT\tWW\"\n")
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`AT611904300234573201`,
		`BKAUATWW`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceAcceptsGibraltarEligibleEPCQRCode(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: GI75NWBK000000007099453\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error for Gibraltar EPC QR code: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`GI75NWBK000000007099453`,
		`\qrcode{\invoxqrcodepayload}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceUsesUTF8EPCQRCodeOverrides(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  payment_terms_text: Pay within 30 days\n", ""+
		"  payment_terms_text: Pay within 30 days\n"+
		"  epc_qr:\n"+
		"    name: \"Boris Österreich & Co.\"\n"+
		"    purpose: gdDs\n"+
		"    text: \"Invoice CUST-001-001 & Überweisung\"\n"+
		"    information: \"Grüße €\"\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`Boris ^^c3^^96sterreich \noexpand\& Co.`,
		`Invoice CUST-001-001 \noexpand\& ^^c3^^9cberweisung`,
		`Gr^^c3^^bc^^c3^^9fe ^^e2^^82^^ac}`,
		`\noexpand\?GDDS\noexpand\?\noexpand\?`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceLeavesEPCQRCodeEmptyWhenInvoiceIsSettled(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0\n", "  paid_amount: 252\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\nBefore\n@@EPC_QR_CODE@@\nAfter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	if strings.Contains(text, `\qrcode{`) {
		t.Fatalf("rendered output %q unexpectedly contains a QR code", text)
	}
	if !strings.Contains(text, "Before\n\nAfter") {
		t.Fatalf("rendered output %q does not show the empty placeholder expansion", text)
	}
}

func TestRenderInvoiceLeavesEPCQRCodeEmptyForNonEURInvoices(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(customersPath)
	if err != nil {
		t.Fatalf("ReadFile(customersPath) returned error: %v", err)
	}
	mutated := strings.TrimSpace(string(source)) + "\n  billing:\n    currency: USD\n"
	if err := os.WriteFile(customersPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customersPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\nBefore\n@@EPC_QR_CODE@@\nAfter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error for non-EUR EPC QR code: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	if strings.Contains(text, `\qrcode{`) {
		t.Fatalf("rendered output %q unexpectedly contains a QR code", text)
	}
	if !strings.Contains(text, "Before\n\nAfter") {
		t.Fatalf("rendered output %q does not show the empty placeholder expansion", text)
	}
}

func TestRenderInvoiceSkipsEPCValidationWhenPlaceholderIsUnused(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("Invoice @@INVOICE_NUMBER@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error without EPC placeholder: %v", err)
	}
}

func TestRenderInvoiceLeavesEPCQRAvailabilityAndLabelInactiveWithoutQRCodePlaceholder(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\usepackage{qrcode}
Before
@@EPC_QR_AVAILABLE@@
@@EPC_QR_LABEL@@
After
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error when only the EPC QR label is active: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, unwanted := range []string{
		`Pay via EPC-QR`,
		`\qrcode{`,
		"\n1\n",
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered output %q unexpectedly contains %q", text, unwanted)
		}
	}
	if !strings.Contains(text, "Before\n0\n\nAfter") {
		t.Fatalf("rendered output %q does not show the inactive EPC QR flag and empty label without a QR placeholder", text)
	}
}

func TestRenderInvoiceRejectsInvalidEligibleEPCQRCode(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for invalid eligible EPC QR data")
	}
	if !strings.Contains(err.Error(), "issuer.payment.iban: invalid IBAN `INVALID`") {
		t.Fatalf("error %q does not contain the invalid IBAN message", err.Error())
	}
}

func TestRenderInvoiceRejectsNonSEPAEligibleEPCQRCode(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(issuerPath)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: BR150000000000000000000000000\n", 1)
	if err := os.WriteFile(issuerPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for non-SEPA eligible EPC QR data")
	}
	if !strings.Contains(err.Error(), "outside the current SEPA scheme scope") {
		t.Fatalf("error %q does not contain the non-SEPA IBAN message", err.Error())
	}
}

func TestIsValidIBANRejectsUnknownCountryCodeAndWrongLength(t *testing.T) {
	tests := []struct {
		name  string
		iban  string
		valid bool
	}{
		{name: "valid Austria", iban: "AT611904300234573201", valid: true},
		{name: "valid Poland", iban: "PL61109010140000071219812874", valid: true},
		{name: "unknown country code", iban: "ZZ6600000000000", valid: false},
		{name: "wrong Austria length", iban: "AT61190430023457320", valid: false},
		{name: "non-numeric check digits", iban: "ATAA1904300234573201", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIBAN(tt.iban); got != tt.valid {
				t.Fatalf("isValidIBAN(%q) = %v, want %v", tt.iban, got, tt.valid)
			}
		})
	}
}

func TestIsSEPASchemeIBAN(t *testing.T) {
	tests := []struct {
		name  string
		iban  string
		valid bool
	}{
		{name: "sepa Austria", iban: "AT611904300234573201", valid: true},
		{name: "sepa GB prefix covers Crown Dependencies", iban: "GB29NWBK60161331926819", valid: true},
		{name: "sepa Gibraltar", iban: "GI75NWBK000000007099453", valid: true},
		{name: "sepa Poland", iban: "PL61109010140000071219812874", valid: true},
		{name: "non-sepa Brazil", iban: "BR150000000000000000000000000", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSEPASchemeIBAN(tt.iban); got != tt.valid {
				t.Fatalf("isSEPASchemeIBAN(%q) = %v, want %v", tt.iban, got, tt.valid)
			}
		})
	}
}

func TestSEPASchemeIBANCountryCodesMatchCurrentEPCIBANCodeList(t *testing.T) {
	expected := []string{
		"AD", "AL", "AT", "BE", "BG", "CH", "CY", "CZ", "DE", "DK",
		"EE", "ES", "FI", "FR", "GB", "GI", "GR", "HR", "HU", "IE",
		"IS", "IT", "LI", "LT", "LU", "LV", "MC", "MD", "ME", "MK",
		"MT", "NL", "NO", "PL", "PT", "RO", "RS", "SE", "SI", "SK",
		"SM", "VA",
	}

	if got, want := len(sepaSchemeIBANCountryCodes), len(expected); got != want {
		t.Fatalf("len(sepaSchemeIBANCountryCodes) = %d, want %d", got, want)
	}
	for _, code := range expected {
		if _, ok := sepaSchemeIBANCountryCodes[code]; !ok {
			t.Fatalf("sepaSchemeIBANCountryCodes is missing %q", code)
		}
	}
}

func TestSEPASchemeIBANCountryCodesHaveIBANLengthDefinitions(t *testing.T) {
	for code := range sepaSchemeIBANCountryCodes {
		if _, ok := ibanCountryLengths[code]; !ok {
			t.Fatalf("ibanCountryLengths is missing SEPA IBAN country code %q", code)
		}
	}
}

func TestStarterTemplateOmitsEPCSectionForNonEURInvoices(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(customersPath)
	if err != nil {
		t.Fatalf("ReadFile(customersPath) returned error: %v", err)
	}
	mutated := strings.TrimSpace(string(source)) + "\n  billing:\n    currency: USD\n"
	if err := os.WriteFile(customersPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customersPath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templateSource, err := starterFiles.ReadFile("starter/template.tex")
	if err != nil {
		t.Fatalf("ReadFile(starter/template.tex) returned error: %v", err)
	}
	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, templateSource, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error for non-EUR starter template: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	if !strings.Contains(text, `\ifnum0=1`) {
		t.Fatalf("rendered starter template %q does not contain the inactive EPC QR conditional", text)
	}
	for _, unwanted := range []string{
		"Pay via EPC-QR",
		`\\qrcode{`,
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered starter template %q unexpectedly contains %q", text, unwanted)
		}
	}
}

func TestStarterTemplateOmitsEPCSectionForSettledInvoices(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0\n", "  paid_amount: 252\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templateSource, err := starterFiles.ReadFile("starter/template.tex")
	if err != nil {
		t.Fatalf("ReadFile(starter/template.tex) returned error: %v", err)
	}
	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, templateSource, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error for settled starter template: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	if !strings.Contains(text, `\ifnum0=1`) {
		t.Fatalf("rendered starter template %q does not contain the inactive EPC QR conditional", text)
	}
	for _, unwanted := range []string{
		"Pay via EPC-QR",
		`\\qrcode{`,
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered starter template %q unexpectedly contains %q", text, unwanted)
		}
	}
}

func TestRenderInvoiceMigratesLegacyStarterVATRow(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "invoice_template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\begin{tabular}{lr}
Subtotal: & @@SUBTOTAL@@\\
VAT (@@VAT_RATE@@\%): & @@VAT_AMOUNT@@\\
Total: & @@TOTAL@@\\
\end{tabular}
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	if strings.Contains(text, "@@VAT_RATE@@") || strings.Contains(text, "@@VAT_AMOUNT@@") {
		t.Fatalf("rendered output still contains legacy VAT placeholders: %q", text)
	}
	if !strings.Contains(text, "VAT (20\\%): & 42,00 \\euro\\\\") {
		t.Fatalf("rendered output %q does not contain migrated VAT summary row", text)
	}
}

func TestRenderInvoiceRejectsLegacyVATPlaceholders(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("VAT @@VAT_RATE@@ @@VAT_AMOUNT@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for legacy VAT placeholders")
	}
	for _, want := range []string{
		"@@VAT_RATE@@: unsupported placeholder; use @@VAT_SUMMARY_ROWS@@",
		"@@VAT_AMOUNT@@: unsupported placeholder; use @@VAT_SUMMARY_ROWS@@",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestRenderInvoiceRejectsLegacyCityAndPostalCodePlaceholders(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@ @@CUSTOMER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for legacy city/postal placeholders")
	}
	for _, want := range []string{
		"@@ISSUER_CITY_AND_POSTAL_CODE@@: unsupported placeholder; use @@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@",
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@: unsupported placeholder; use @@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestRenderInvoiceRejectsUnmatchedLineItemBlockPlaceholders(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@LINE_ITEMS_BEGIN@@\n@@LINE_ITEM_NAME@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for unmatched line-item block placeholder")
	}
	want := "@@LINE_ITEMS_BEGIN@@: missing matching @@LINE_ITEMS_END@@"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

func TestRenderInvoiceRejectsLineItemPlaceholdersOutsideCustomBlock(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("@@LINE_ITEM_NAME@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = RenderInvoice(templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for line-item placeholder outside custom block")
	}
	want := "@@LINE_ITEM_NAME@@: only supported inside @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

func TestRenderInvoiceCopiesTemplateAssetsToOutputDir(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(
		customersPath,
		issuerPath,
		invoicePath,
	)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "invoice.tex")
	if err := RenderInvoice(templatePath, outputPath, ctx); err != nil {
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

func TestCreateInvoiceEmailDraftIncludesAttachmentAndHeaders(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config-home"))

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := CreateInvoiceEmailDraft(customersPath, issuerPath, invoicePath, pdfPath, outputPath, "", "")
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.OutputPath != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.OutputPath, outputPath)
	}
	if draft.Recipient != "office@appsters.example" {
		t.Fatalf("Recipient = %q, want %q", draft.Recipient, "office@appsters.example")
	}
	if draft.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", draft.Subject, "Invoice CUST-001-001")
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(eml)
	for _, want := range []string{
		"hello@example.com",
		"office@appsters.example",
		"Subject: Invoice CUST-001-001",
		"X-Unsent: 1",
		`filename="invoice.pdf"`,
		"Dear Jane Doe,",
		"Please find attached invoice CUST-001-001.",
		"Outstanding amount: 252,00 EUR",
		"Regards,\r\nBoris Consulting\r\n\r\n\r\n--invox-boundary-",
		"JVBERi0xLjQKZmFrZQ==",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("draft email does not contain %q:\n%s", want, text)
		}
	}
}

func TestCreateInvoiceEmailDraftAllowsArchivedInvoiceStatus(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: archived", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := CreateInvoiceEmailDraft(customersPath, issuerPath, invoicePath, pdfPath, outputPath, "", "")
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}
	if draft.OutputPath != outputPath {
		t.Fatalf("OutputPath = %q, want %q", draft.OutputPath, outputPath)
	}
}

func TestCreateInvoiceEmailDraftUsesConfiguredBodyTemplate(t *testing.T) {
	writeConfigFile(t, strings.TrimSpace(`
email:
  body: |
    {email_greeting}

    Invoice {invoice_number} is due on {due_date}.
    Open amount: {outstanding_amount}
    Terms: {payment_terms_text}

    Regards,
    {issuer_name}
`)+"\n")

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	if _, err := CreateInvoiceEmailDraft(customersPath, issuerPath, invoicePath, pdfPath, outputPath, "", ""); err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(eml)
	for _, want := range []string{
		"Dear Jane Doe,",
		"Invoice CUST-001-001 is due on 2026-04-05.",
		"Open amount: 252,00 EUR",
		"Terms: Pay within 30 days",
		"Regards,",
		"Regards,\r\nBoris Consulting\r\n\r\n\r\n--invox-boundary-",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("draft email does not contain %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Please find attached invoice CUST-001-001.") {
		t.Fatalf("draft email should not contain the default body:\n%s", text)
	}
}

func TestCreateInvoiceEmailDraftUsesConfiguredSubjectTemplateWithAllPlaceholders(t *testing.T) {
	writeConfigFile(t, strings.TrimSpace(`
email:
  subject: "{customer_name} | {email_greeting} | {contact_person} | {customer_id} | {invoice_number} | {issue_date} | {due_date} | {total_amount} | {outstanding_amount} | {payment_terms_text} | {issuer_name}"
`)+"\n")

	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := CreateInvoiceEmailDraft(customersPath, issuerPath, invoicePath, pdfPath, outputPath, "", "")
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	wantSubject := "Appsters GmbH | Dear Jane Doe, | Jane Doe | CUST-001 | CUST-001-001 | 2026-03-06 | 2026-04-05 | 252,00 EUR | 252,00 EUR | Pay within 30 days | Boris Consulting"
	if draft.Subject != wantSubject {
		t.Fatalf("Subject = %q, want %q", draft.Subject, wantSubject)
	}

	eml, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	wantHeader := "Subject: " + mime.QEncoding.Encode("utf-8", wantSubject)
	if !strings.Contains(string(eml), wantHeader) {
		t.Fatalf("draft email does not contain %q:\n%s", wantHeader, string(eml))
	}
}

func TestCreateInvoiceEmailDraftExpandsSubjectOverridePlaceholders(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")

	draft, err := CreateInvoiceEmailDraft(
		customersPath,
		issuerPath,
		invoicePath,
		pdfPath,
		outputPath,
		"",
		"Invoice {invoice_number} for {contact_person}",
	)
	if err != nil {
		t.Fatalf("CreateInvoiceEmailDraft returned error: %v", err)
	}

	if draft.Subject != "Invoice CUST-001-001 for Jane Doe" {
		t.Fatalf("Subject = %q, want %q", draft.Subject, "Invoice CUST-001-001 for Jane Doe")
	}
}

func TestCreateInvoiceEmailDraftRejectsInvoiceWithoutSendableStatus(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	_, err := CreateInvoiceEmailDraft(
		customersPath,
		issuerPath,
		invoicePath,
		pdfPath,
		filepath.Join(t.TempDir(), "invoice.eml"),
		"",
		"",
	)
	if err == nil {
		t.Fatal("CreateInvoiceEmailDraft returned nil error for non-built invoice")
	}
	if !strings.Contains(err.Error(), "invoice.status must be `built` or `archived` before creating an email draft") {
		t.Fatalf("error %q does not contain sendable status validation", err.Error())
	}
}

func TestResolveEmailDraftPaths(t *testing.T) {
	rootDir := t.TempDir()
	yamlInput := filepath.Join(rootDir, "BL00210001.yaml")
	pdfInput := filepath.Join(rootDir, "BL00210001.pdf")
	if err := os.WriteFile(yamlInput, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(yamlInput) returned error: %v", err)
	}

	tests := []struct {
		name          string
		inputPath     string
		pdfPath       string
		outputPath    string
		wantInvoice   string
		wantPDF       string
		wantOutput    string
		wantErrSubstr string
	}{
		{
			name:        "yaml input derives sibling pdf and eml",
			inputPath:   yamlInput,
			wantInvoice: yamlInput,
			wantPDF:     pdfInput,
			wantOutput:  filepath.Join(rootDir, "BL00210001.eml"),
		},
		{
			name:        "pdf input derives sibling yaml and eml",
			inputPath:   pdfInput,
			wantInvoice: yamlInput,
			wantPDF:     pdfInput,
			wantOutput:  filepath.Join(rootDir, "BL00210001.eml"),
		},
		{
			name:        "explicit overrides are preserved",
			inputPath:   pdfInput,
			pdfPath:     filepath.Join(rootDir, "outgoing.pdf"),
			outputPath:  filepath.Join(rootDir, "drafts", "outgoing.eml"),
			wantInvoice: yamlInput,
			wantPDF:     filepath.Join(rootDir, "outgoing.pdf"),
			wantOutput:  filepath.Join(rootDir, "drafts", "outgoing.eml"),
		},
		{
			name:          "unsupported input extension",
			inputPath:     filepath.Join(rootDir, "BL00210001.txt"),
			wantErrSubstr: "input must end with .yaml, .yml, or .pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := ResolveEmailDraftPaths(tt.inputPath, tt.pdfPath, tt.outputPath)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("ResolveEmailDraftPaths returned nil error")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
			}
			if paths.InvoicePath != tt.wantInvoice {
				t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, tt.wantInvoice)
			}
			if paths.PDFPath != tt.wantPDF {
				t.Fatalf("PDFPath = %q, want %q", paths.PDFPath, tt.wantPDF)
			}
			if paths.OutputPath != tt.wantOutput {
				t.Fatalf("OutputPath = %q, want %q", paths.OutputPath, tt.wantOutput)
			}
		})
	}
}

func TestResolveEmailDraftPathsFallsBackToArchiveDir(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedInvoicePath := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedInvoicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(archivedInvoicePath)) returned error: %v", err)
	}
	if err := os.WriteFile(archivedInvoicePath, []byte("invoice:\n  number: BL00210001\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedInvoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "BL00210001.pdf")
	paths, err := ResolveEmailDraftPaths(pdfPath, "", "")
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if paths.InvoicePath != archivedInvoicePath {
		t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, archivedInvoicePath)
	}
	if paths.PDFPath != pdfPath {
		t.Fatalf("PDFPath = %q, want %q", paths.PDFPath, pdfPath)
	}
	if paths.OutputPath != filepath.Join(filepath.Dir(pdfPath), "BL00210001.eml") {
		t.Fatalf("OutputPath = %q, want %q", paths.OutputPath, filepath.Join(filepath.Dir(pdfPath), "BL00210001.eml"))
	}
}

func TestResolveEmailDraftPathsRejectsAmbiguousArchiveMatches(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

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

	_, err := ResolveEmailDraftPaths(filepath.Join(t.TempDir(), "BL00210001.pdf"), "", "")
	if err == nil {
		t.Fatal("ResolveEmailDraftPaths returned nil error for ambiguous archive matches")
	}
	if !strings.Contains(err.Error(), "multiple archived invoice YAML files match") {
		t.Fatalf("error %q does not contain ambiguity message", err.Error())
	}
}

func TestCopyTemplateAssetsFallsBackToGlobalConfig(t *testing.T) {
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

	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.WriteFile(filepath.Join(configDir, "logo.png"), []byte("logo"), 0o644); err != nil {
		t.Fatalf("WriteFile(global logo) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "fonts", "Ubuntu-Regular.ttf"), []byte("font"), 0o644); err != nil {
		t.Fatalf("WriteFile(global font) returned error: %v", err)
	}

	templatePath := filepath.Join(templateDir, "invoice_template.tex")
	outputPath := filepath.Join(outputDir, "invoice.tex")
	rendered := "\\setmainfont{Ubuntu}[Path=fonts/,UprightFont=Ubuntu-Regular.ttf]\n\\includegraphics{logo.png}\n"

	if err := copyTemplateAssets(templatePath, outputPath, rendered); err != nil {
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

func TestLoadContextRejectsOverpaidInvoice(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoice.yaml) returned error: %v", err)
	}

	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 999999", 1)
	mutatedPath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(mutatedPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	_, err = LoadContext(
		customersPath,
		issuerPath,
		mutatedPath,
	)
	if err == nil {
		t.Fatalf("LoadContext returned nil error for overpaid invoice")
	}
	if !strings.Contains(err.Error(), "exceeds total") {
		t.Fatalf("error %q does not contain %q", err.Error(), "exceeds total")
	}
}

func TestDefaultOptionsPreferLocalProjectFilesOverGlobalConfig(t *testing.T) {
	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(configDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}
	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(workDir, name)
		if err := os.WriteFile(path, []byte("local"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
		path = filepath.Join(customDir, name)
		if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	configSource := strings.TrimSpace(`
paths:
  customers: ../../custom/customers.yaml
  issuer: ../../custom/issuer.yaml
  defaults: ../../custom/invoice_defaults.yaml
  template: ../../custom/template.tex
`) + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configSource), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	chdirForTest(t, workDir)

	opts, err := DefaultOptions()
	if err != nil {
		t.Fatalf("DefaultOptions returned error: %v", err)
	}

	if opts.BaseDir != workDir {
		t.Fatalf("BaseDir = %q, want %q", opts.BaseDir, workDir)
	}
	if opts.CustomersPath != filepath.Join(workDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want local project path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(workDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want local project path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(workDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want local project path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(workDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want local project path", opts.TemplatePath)
	}
}

func TestDefaultOptionsFallbackToGlobalConfigFiles(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(configDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(defaultConfigTemplate()), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	chdirForTest(t, workDir)

	opts, err := DefaultOptions()
	if err != nil {
		t.Fatalf("DefaultOptions returned error: %v", err)
	}

	if opts.CustomersPath != filepath.Join(configDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want global config path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(configDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want global config path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(configDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want global config path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(configDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want global config path", opts.TemplatePath)
	}
}

func TestDefaultOptionsUseConfiguredPathsBeforeGlobalDefaults(t *testing.T) {
	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		if err := os.WriteFile(filepath.Join(customDir, name), []byte("custom"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(configDir, name), []byte("global"), 0o644); err != nil {
			t.Fatalf("WriteFile(global %s) returned error: %v", name, err)
		}
	}

	configSource := strings.TrimSpace(`
paths:
  customers: ../../custom/customers.yaml
  issuer: ../../custom/issuer.yaml
  defaults: ../../custom/invoice_defaults.yaml
  template: ../../custom/template.tex
`) + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configSource), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	chdirForTest(t, workDir)

	opts, err := DefaultOptions()
	if err != nil {
		t.Fatalf("DefaultOptions returned error: %v", err)
	}

	if opts.CustomersPath != filepath.Join(customDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want configured path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(customDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want configured path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(customDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want configured path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(customDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want configured path", opts.TemplatePath)
	}
}

func TestListTemplatesUsesDefaultTemplateDirectoryOnly(t *testing.T) {
	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, file := range []string{
		filepath.Join(workDir, "project.tex"),
		filepath.Join(configDir, "template.tex"),
		filepath.Join(customDir, "custom.tex"),
		filepath.Join(customDir, "multi_vat.tex"),
	} {
		if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", file, err)
		}
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	templates, err := ListTemplates(workDir)
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}

	got := make([]string, 0, len(templates))
	for _, template := range templates {
		got = append(got, template.Name+"\t"+template.Path)
	}
	for _, want := range []string{
		"custom.tex\t" + filepath.Join(customDir, "custom.tex"),
		"multi_vat.tex\t" + filepath.Join(customDir, "multi_vat.tex"),
	} {
		if !containsString(got, want) {
			t.Fatalf("template list %q does not contain %q", got, want)
		}
	}
	for _, forbidden := range []string{
		"project.tex\t" + filepath.Join(workDir, "project.tex"),
		"template.tex\t" + filepath.Join(configDir, "template.tex"),
	} {
		if containsString(got, forbidden) {
			t.Fatalf("template list %q should not contain %q", got, forbidden)
		}
	}
}

func TestResolveTemplateReferenceFindsNamedConfigTemplate(t *testing.T) {
	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	templatePath := filepath.Join(customDir, "multi_vat.tex")
	if err := os.WriteFile(templatePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "custom.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(custom.tex) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	resolvedPath, err := ResolveTemplateReference(workDir, "multi_vat.tex")
	if err != nil {
		t.Fatalf("ResolveTemplateReference returned error: %v", err)
	}
	if resolvedPath != templatePath {
		t.Fatalf("resolvedPath = %q, want %q", resolvedPath, templatePath)
	}
}

func TestResolveTemplateReferenceDoesNotSearchOutsideDefaultTemplateDirectory(t *testing.T) {
	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "custom.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(custom.tex) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "outside.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside.tex) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	_, err := ResolveTemplateReference(workDir, "outside.tex")
	if err == nil {
		t.Fatal("ResolveTemplateReference returned nil error for template outside default template directory")
	}
	if !strings.Contains(err.Error(), "template \"outside.tex\" not found") {
		t.Fatalf("error %q does not contain not-found message", err.Error())
	}
}

func TestDefaultOptionsFallbackToLegacyConfigFiles(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	legacyDir := filepath.Join(configHome, "invoice-tool")
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(legacyDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	chdirForTest(t, workDir)

	opts, err := DefaultOptions()
	if err != nil {
		t.Fatalf("DefaultOptions returned error: %v", err)
	}

	if opts.CustomersPath != filepath.Join(legacyDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want legacy config path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(legacyDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want legacy config path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(legacyDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want legacy config path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(legacyDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want legacy config path", opts.TemplatePath)
	}
}

func TestResolveArchiveDirDefaultsToPlatformDataDir(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(defaultConfigTemplate()), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	var expected string
	switch runtime.GOOS {
	case "darwin":
		expected = filepath.Join(homeDir, "Library", "Application Support", "invox", "invoices")
	case "windows":
		appData := filepath.Join(homeDir, "AppData", "Roaming")
		t.Setenv("APPDATA", appData)
		expected = filepath.Join(appData, "invox", "invoices")
	default:
		dataHome := filepath.Join(homeDir, ".local", "share")
		t.Setenv("XDG_DATA_HOME", "")
		expected = filepath.Join(dataHome, "invox", "invoices")
	}

	got, err := ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, expected)
	}
}

func TestResolveArchiveDirUsesConfigOverride(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("archive:\n  dir: ~/Documents/Invox/Invoices\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	got, err := ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(homeDir, "Documents", "Invox", "Invoices")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestResolveArchiveDirUsesRelativeConfigPath(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("archive:\n  dir: archive\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	got, err := ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(configDir, "archive")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestResolveArchiveDirUsesLegacyConfigOverride(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	legacyDir := filepath.Join(configHome, "invoice-tool")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("archive:\n  dir: archived-invoices\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	got, err := ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(legacyDir, "archived-invoices")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestEditableConfigPathCreatesCommentedTemplate(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	path, err := EditableConfigPath()
	if err != nil {
		t.Fatalf("EditableConfigPath returned error: %v", err)
	}

	wantPath := filepath.Join(configHome, "invox", "config.yaml")
	if path != wantPath {
		t.Fatalf("EditableConfigPath() = %q, want %q", path, wantPath)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}

	text := string(source)
	for _, want := range []string{
		"# Invox user configuration.",
		"# Supported settings:",
		"# - Top-level keys must not be indented.",
		"#   paths.customers",
		"#   paths.issuer",
		"#   paths.defaults",
		"#   paths.template",
		"#   numbering.pattern",
		"#   numbering.start",
		"#   archive.dir",
		"#   email.subject",
		"#   email.body",
		"#       {email_greeting}",
		"#       {contact_person}",
		"# - Per-customer numbering overrides live in customers.yaml at:",
		"#   <customer>.numbering.start",
		"# paths:",
		"#   customers: 'customers.yaml'",
		"#   issuer: 'issuer.yaml'",
		"#   defaults: 'invoice_defaults.yaml'",
		"#   template: 'template.tex'",
		"# numbering:",
		"#   pattern: '{customer_code}-{counter:03}'",
		"#   start: 1",
		"# archive:",
		"#   dir: '~/Library/Application Support/invox/invoices'",
		"# email:",
		"#   subject: 'Invoice {invoice_number}'",
		"#   body: |",
		"#     Please find attached invoice {invoice_number}.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config template %q does not contain %q", text, want)
		}
	}
}

func TestEditableConfigPathPreservesExistingConfig(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", configHome)

	want := "archive:\n  dir: ~/Documents/Invox/Invoices\n"
	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	gotPath, err := EditableConfigPath()
	if err != nil {
		t.Fatalf("EditableConfigPath returned error: %v", err)
	}
	if gotPath != path {
		t.Fatalf("EditableConfigPath() = %q, want %q", gotPath, path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("config.yaml = %q, want %q", string(got), want)
	}
}

func TestResolveDefaultCustomersPathRejectsIndentedTopLevelConfig(t *testing.T) {
	writeConfigFile(t, " numbering:\n  pattern: '{customer_id}-{counter:03}'\npaths:\n  customers: '~/customers.yaml'\n")

	_, err := ResolveDefaultCustomersPath(t.TempDir())
	if err == nil {
		t.Fatalf("ResolveDefaultCustomersPath returned nil error for indented top-level config")
	}
	if !strings.Contains(err.Error(), "top-level keys must not be indented") {
		t.Fatalf("error %q does not contain top-level indentation message", err.Error())
	}
}

func TestCreateNewInvoicePrefillsDatesAndNumber(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeArchivedInvoiceMarkdown(t, archiveDir, "2026-03-05.md", "CUST-001-001")
	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")

	invoiceNumber, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if invoiceNumber != "CUST-001-002" {
		t.Fatalf("invoiceNumber = %q, want %q", invoiceNumber, "CUST-001-002")
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	for _, want := range []string{
		"customer_id: CUST-001",
		"number: CUST-001-002",
		"issue_date: \"2026-03-06\"",
		"due_date: \"2026-04-05\"",
		"period: \"Leistungszeitraum: \"",
		"vat_percent: 20",
		"paid_amount: \"0\"",
		"name: Beispielposition",
		"description: Beschreibung der Leistung",
		"unit_price: 100",
		"quantity: 1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "payment_terms_text:") {
		t.Fatalf("created invoice should not contain payment_terms_text anymore:\n%s", text)
	}
	for _, forbidden := range []string{"period_label:", "vat_rate_percent:", "line_items:"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("created invoice should not contain legacy key %q:\n%s", forbidden, text)
		}
	}
}

func TestCreateNewInvoicePrefillsVATFromCustomerDefaultWhenMissing(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  tax:
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(defaultsPath, []byte(strings.TrimSpace(`
invoice:
  period: "Leistungszeitraum: "
positions:
  - name: Beispielposition
    description: Beschreibung der Leistung
    unit_price: 100
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice_defaults.yaml) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, _, err := CreateNewInvoice(defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", false)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if !strings.Contains(string(source), "vat_percent: \"13\"") {
		t.Fatalf("created invoice does not contain customer VAT default:\n%s", string(source))
	}
}

func TestCreateNewInvoicePreservesSourceVATWhenCustomerHasDefault(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  tax:
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, _, err := CreateNewInvoice(defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", false)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	if !strings.Contains(text, "vat_percent: 20") {
		t.Fatalf("created invoice does not preserve source VAT:\n%s", text)
	}
	if strings.Contains(text, "vat_percent: \"13\"") {
		t.Fatalf("created invoice should not overwrite source VAT with customer default:\n%s", text)
	}
}

func TestCreateNewInvoiceStartsFromConfiguredStartWhenArchiveHasNoMatch(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 7\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	invoiceNumber, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if invoiceNumber != "CUST-001-007" {
		t.Fatalf("invoiceNumber = %q, want %q", invoiceNumber, "CUST-001-007")
	}
}

func TestCreateNewInvoiceFailsWhenArchiveContainsInvalidFrontMatter(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-05.md")
	if err := os.WriteFile(archivePath, []byte(strings.Join([]string{
		"---",
		"invoice:",
		"  number: CUST-001-010",
		"  broken: [",
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", archivePath, err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for invalid archived invoice front matter")
	}
	if !strings.Contains(err.Error(), archivePath) {
		t.Fatalf("error %q does not contain archived invoice path %q", err.Error(), archivePath)
	}
	if !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("error %q does not contain YAML parse context", err.Error())
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file should not be created, stat error = %v", statErr)
	}
}

func TestCreateNewInvoiceRejectsLegacyDefaultKeys(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeLegacyDraftFixtures(t)
	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")

	_, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for legacy default keys")
	}
	for _, want := range []string{
		defaultsPath + ": invoice.period_label: unsupported key; use invoice.period",
		defaultsPath + ": invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		defaultsPath + ": line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestCreateNewInvoiceUsesCustomerSpecificStartWhenArchiveHasNoMatch(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixturesWithCustomerStart(t, "7")
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	invoiceNumber, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if invoiceNumber != "CUST-001-007" {
		t.Fatalf("invoiceNumber = %q, want %q", invoiceNumber, "CUST-001-007")
	}
}

func TestCreateNewInvoiceFromLastArchivedInvoice(t *testing.T) {
	oldCurrentDate := currentDate
	currentDate = func() time.Time {
		return time.Date(2026, 3, 10, 12, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		currentDate = oldCurrentDate
	})

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	olderArchivePath := filepath.Join(archiveDir, "2026-03-01.yaml")
	if err := os.WriteFile(olderArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-01
  due_date: 2026-03-31
  status: archived
  period: February 2026
  vat_percent: 19
  paid_amount: 500
positions:
  - name: Older position
    description: From older invoice
    unit_price: 50
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(olderArchivePath) returned error: %v", err)
	}

	latestArchivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(latestArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period: March 2026
  vat_percent: 10
  paid_amount: 999
positions:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(latestArchivePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	invoiceNumber, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		true,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if invoiceNumber != "CUST-001-003" {
		t.Fatalf("invoiceNumber = %q, want %q", invoiceNumber, "CUST-001-003")
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	for _, want := range []string{
		"number: CUST-001-003",
		"issue_date: \"2026-03-10\"",
		"due_date: \"2026-04-09\"",
		"status: draft",
		"paid_amount: \"0\"",
		"period: March 2026",
		"vat_percent: 10",
		"name: Latest position",
		"description: From latest invoice",
		"unit_price: 120",
		"quantity: 2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{
		"Older position",
		"From older invoice",
		"status: archived",
		"paid_amount: 999",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("created invoice should not contain %q:\n%s", forbidden, text)
		}
	}
}

func TestCreateNewInvoiceFromLastRequiresArchivedInvoiceForCustomer(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, _, err := CreateNewInvoice(
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		true,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error without archived invoice match")
	}
	if !strings.Contains(err.Error(), "no archived invoice found for customer_id `CUST-001`") {
		t.Fatalf("error %q does not contain missing archived invoice message", err.Error())
	}
}

func TestCreateNewInvoiceFromLastRejectsLegacyArchivedInvoiceKeys(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period_label: March 2026
  vat_rate_percent: 10
  paid_amount: 999
line_items:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, _, err := CreateNewInvoice(defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", true)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for legacy archived invoice keys")
	}
	for _, want := range []string{
		archivePath + ": invoice.period_label: unsupported key; use invoice.period",
		archivePath + ": invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		archivePath + ": line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestEditArchivedMarkdownInvoiceAndRearchiveAsYAML(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedMarkdownPath := filepath.Join(archiveDir, "customer-a", "2026-03-06.md")
	if err := os.MkdirAll(filepath.Dir(archivedMarkdownPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive subdir) returned error: %v", err)
	}
	if err := os.WriteFile(archivedMarkdownPath, []byte(strings.Join([]string{
		"---",
		"customer_id: CUST-001",
		"invoice:",
		"  number: CUST-001-001",
		"  issue_date: 2026-03-06",
		"  due_date: 2026-04-05",
		"  status: archived",
		"  period: March 2026",
		"  vat_percent: 20",
		"  paid_amount: 0",
		"positions:",
		"  - name: Development",
		"    description: Markdown archive",
		"    unit_price: 100",
		"    quantity: 2",
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedMarkdownPath) returned error: %v", err)
	}

	workDir := t.TempDir()
	outputPath, archivePath, err := EditArchivedInvoice("customer-a/2026-03-06.md", workDir)
	if err != nil {
		t.Fatalf("EditArchivedInvoice returned error: %v", err)
	}
	if archivePath != archivedMarkdownPath {
		t.Fatalf("archivePath = %q, want %q", archivePath, archivedMarkdownPath)
	}

	wantOutputPath := filepath.Join(workDir, "2026-03-06.yaml")
	if outputPath != wantOutputPath {
		t.Fatalf("outputPath = %q, want %q", outputPath, wantOutputPath)
	}
	editedSource, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	editedText := string(editedSource)
	for _, want := range []string{
		"status: editing",
		"archive_path: customer-a/2026-03-06.yaml",
		"archive_replace_path: customer-a/2026-03-06.md",
		"Markdown archive",
	} {
		if !strings.Contains(editedText, want) {
			t.Fatalf("edited invoice does not contain %q:\n%s", want, editedText)
		}
	}

	mutated := strings.Replace(editedText, "Markdown archive", "Edited markdown archive", 1)
	if err := os.WriteFile(outputPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	finalArchivePath, err := ArchiveInvoice(outputPath)
	if err != nil {
		t.Fatalf("ArchiveInvoice returned error: %v", err)
	}
	wantArchivePath := filepath.Join(archiveDir, "customer-a", "2026-03-06.yaml")
	if finalArchivePath != wantArchivePath {
		t.Fatalf("finalArchivePath = %q, want %q", finalArchivePath, wantArchivePath)
	}
	if _, err := os.Stat(archivedMarkdownPath); err == nil {
		t.Fatalf("legacy markdown archive should have been removed: %s", archivedMarkdownPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(archivedMarkdownPath) returned unexpected error: %v", err)
	}
	if _, err := os.Stat(outputPath); err == nil {
		t.Fatalf("working copy should have been removed: %s", outputPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(outputPath) returned unexpected error: %v", err)
	}

	finalSource, err := os.ReadFile(finalArchivePath)
	if err != nil {
		t.Fatalf("ReadFile(finalArchivePath) returned error: %v", err)
	}
	finalText := string(finalSource)
	for _, want := range []string{
		"status: archived",
		"Edited markdown archive",
	} {
		if !strings.Contains(finalText, want) {
			t.Fatalf("final archive does not contain %q:\n%s", want, finalText)
		}
	}
	for _, forbidden := range []string{
		"status: editing",
		"_invox:",
	} {
		if strings.Contains(finalText, forbidden) {
			t.Fatalf("final archive should not contain %q:\n%s", forbidden, finalText)
		}
	}
}

func TestEditArchivedInvoiceRejectsLegacyKeys(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-06.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: archived
  period_label: March 2026
  vat_rate_percent: 20
  paid_amount: 0
line_items:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	_, _, err := EditArchivedInvoice("2026-03-06.yaml", t.TempDir())
	if err == nil {
		t.Fatal("EditArchivedInvoice returned nil error for legacy keys")
	}
	for _, want := range []string{
		archivePath + ": invoice.period_label: unsupported key; use invoice.period",
		archivePath + ": invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		archivePath + ": line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestIncrementInvoiceNumberAdvancesCurrentInvoice(t *testing.T) {
	customersPath, _, _ := writeDraftFixtures(t)
	invoicePath := writeMinimalInvoice(t, "CUST-001-009")
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeArchivedInvoiceMarkdown(t, archiveDir, "2026-03-05.md", "CUST-001-011")

	customerID, oldNumber, newNumber, err := IncrementInvoiceNumber(
		invoicePath,
		customersPath,
	)
	if err != nil {
		t.Fatalf("IncrementInvoiceNumber returned error: %v", err)
	}
	if customerID != "CUST-001" {
		t.Fatalf("customerID = %q, want %q", customerID, "CUST-001")
	}
	if oldNumber != "CUST-001-009" {
		t.Fatalf("oldNumber = %q, want %q", oldNumber, "CUST-001-009")
	}
	if newNumber != "CUST-001-012" {
		t.Fatalf("newNumber = %q, want %q", newNumber, "CUST-001-012")
	}

	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(source), "number: CUST-001-012") {
		t.Fatalf("invoice file does not contain the incremented invoice number: %q", string(source))
	}
}

func TestResolveNumberingSettingsUsesConfigAndDefaults(t *testing.T) {
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{year}-{counter:04}'\n  start: 5\n")

	settings, err := ResolveNumberingSettings()
	if err != nil {
		t.Fatalf("ResolveNumberingSettings returned error: %v", err)
	}
	if settings.Pattern != "{customer_id}-{year}-{counter:04}" {
		t.Fatalf("Pattern = %q, want %q", settings.Pattern, "{customer_id}-{year}-{counter:04}")
	}
	if settings.Start != 5 {
		t.Fatalf("Start = %d, want %d", settings.Start, 5)
	}
}

func writeContextFixtures(t *testing.T) (string, string, string, string, string, string) {
	t.Helper()

	dir := t.TempDir()
	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	invoicePath := filepath.Join(dir, "invoice.yaml")
	templatePath := filepath.Join(dir, "invoice_template.tex")
	logoPath := filepath.Join(dir, "logo.png")
	fontPath := filepath.Join(dir, "fonts", "Ubuntu-Regular.ttf")

	if err := os.MkdirAll(filepath.Dir(fontPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(fonts) returned error: %v", err)
	}

	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  status: active
  email: office@appsters.example
  email_greeting: Dear Jane Doe,
  contact_person: Jane Doe
  address:
    street: Hauptstrasse 1
    postal_code: 1010
    city: Vienna
    country: Austria
  tax:
    vat_tax_id: ATU12345678
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	if err := os.WriteFile(issuerPath, []byte(strings.TrimSpace(`
company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: 1010
    city: Vienna
    country: Austria
payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(issuer.yaml) returned error: %v", err)
	}

	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  period: Leistungszeitraum
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
  - name: Support
    description: QA
    unit_price: 10
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
\setmainfont{Ubuntu}[Path=fonts/,UprightFont=Ubuntu-Regular.ttf]
\includegraphics{logo.png}
Invoice @@INVOICE_NUMBER@@
Customer @@CUSTOMER_NAME@@
Terms @@PAYMENT_TERMS_TEXT@@
Rows:
@@LINE_ITEMS_ROWS@@
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice_template.tex) returned error: %v", err)
	}

	if err := os.WriteFile(logoPath, []byte("logo"), 0o644); err != nil {
		t.Fatalf("WriteFile(logo.png) returned error: %v", err)
	}
	if err := os.WriteFile(fontPath, []byte("font"), 0o644); err != nil {
		t.Fatalf("WriteFile(font) returned error: %v", err)
	}

	return customersPath, issuerPath, invoicePath, templatePath, logoPath, fontPath
}

func writeDraftFixtures(t *testing.T) (string, string, string) {
	t.Helper()
	return writeDraftFixturesWithCustomerStart(t, "")
}

func writeDraftFixturesWithCustomerStart(t *testing.T, customerStart string) (string, string, string) {
	t.Helper()

	dir := t.TempDir()
	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	defaultsPath := filepath.Join(dir, "invoice_defaults.yaml")

	customerNumbering := ""
	if strings.TrimSpace(customerStart) != "" {
		customerNumbering = "\n  numbering:\n    start: " + customerStart
	}

	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
`+customerNumbering)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(issuerPath, []byte(strings.TrimSpace(`
payment:
  due_days: 30
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(issuer.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(defaultsPath, []byte(strings.TrimSpace(`
invoice:
  period: "Leistungszeitraum: "
  vat_percent: 20
positions:
  - name: Beispielposition
    description: Beschreibung der Leistung
    unit_price: 100
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice_defaults.yaml) returned error: %v", err)
	}

	return customersPath, issuerPath, defaultsPath
}

func writeLegacyDraftFixtures(t *testing.T) (string, string, string) {
	t.Helper()

	dir := t.TempDir()
	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	defaultsPath := filepath.Join(dir, "invoice_defaults.yaml")

	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(issuerPath, []byte(strings.TrimSpace(`
payment:
  due_days: 30
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(issuer.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(defaultsPath, []byte(strings.TrimSpace(`
invoice:
  period_label: "Leistungszeitraum: "
  vat_rate_percent: 20
line_items:
  - name: Beispielposition
    description: Beschreibung der Leistung
    unit_price: 100
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice_defaults.yaml) returned error: %v", err)
	}

	return customersPath, issuerPath, defaultsPath
}

func writeMinimalInvoice(t *testing.T, invoiceNumber string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "invoice.yaml")
	source := strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: PLACEHOLDER
  issue_date: 2026-03-06
`) + "\n"
	source = strings.Replace(source, "PLACEHOLDER", invoiceNumber, 1)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	return path
}

func writeInvoiceWithoutNumber(t *testing.T, sourcePath string) string {
	t.Helper()

	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(invoice.yaml) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  number: CUST-001-001\n", "", 1)
	if mutated == string(source) {
		t.Fatal("failed to remove invoice.number from the fixture")
	}

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	return invoicePath
}

func writeConfigFile(t *testing.T, source string) string {
	t.Helper()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	return path
}

func writeArchivedInvoiceMarkdown(t *testing.T, dir, name, invoiceNumber string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	source := strings.Join([]string{
		"---",
		"invoice:",
		"  number: " + invoiceNumber,
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
	return path
}

func quoteYAMLString(value string) string {
	return strconv.Quote(value)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
