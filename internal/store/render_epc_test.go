package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderInvoiceRendersEPCQRCode(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`\edef\invoxqrcodepayload{BCD\noexpand\?002\noexpand\?1\noexpand\?SCT\noexpand\?BKAUATWW\noexpand\?Boris\noexpand\ Consulting\noexpand\?AT611904300234573201\noexpand\?EUR252.00`,
		`\noexpand\?\noexpand\?CUST-001-001}`,
		`\qrcode{\invoxqrcodepayload}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}

func TestRenderInvoiceEscapesReservedQRCodeCharactersInDefaultReference(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := isolatedHost(t)
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
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		`Boris\noexpand\ ^^c3^^96sterreich\noexpand\ \noexpand\&\noexpand\ Co.`,
		`Invoice\noexpand\ CUST-001-001\noexpand\ \noexpand\&\noexpand\ ^^c3^^9cberweisung`,
		`Gr^^c3^^bc^^c3^^9fe\noexpand\ ^^e2^^82^^ac}`,
		`\noexpand\?GDDS\noexpand\?\noexpand\?`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output %q does not contain %q", text, want)
		}
	}
}
