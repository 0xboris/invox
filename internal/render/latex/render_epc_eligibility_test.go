package latex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestRenderInvoiceLeavesEPCQRCodeEmptyWhenInvoiceIsSettled(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0\n", "  paid_amount: 252\n", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\nBefore\n@@EPC_QR_CODE@@\nAfter\n"), 0o644); err != nil {
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
	if strings.Contains(text, `\qrcode{`) {
		t.Fatalf("rendered output %q unexpectedly contains a QR code", text)
	}
	if !strings.Contains(text, "Before\n\nAfter") {
		t.Fatalf("rendered output %q does not show the empty placeholder expansion", text)
	}
}

func TestRenderInvoiceLeavesEPCQRCodeEmptyForNonEURInvoices(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Customers)
	if err != nil {
		t.Fatalf("ReadFile(customersPath) returned error: %v", err)
	}
	mutated := strings.TrimSpace(string(source)) + "\n  billing:\n    currency: USD\n"
	if err := os.WriteFile(fx.Customers, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customersPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\nBefore\n@@EPC_QR_CODE@@\nAfter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Issuer)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(fx.Issuer, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("Invoice @@INVOICE_NUMBER@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error without EPC placeholder: %v", err)
	}
}

func TestRenderInvoiceLeavesEPCQRAvailabilityAndLabelInactiveWithoutQRCodePlaceholder(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Issuer)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(fx.Issuer, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
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
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Issuer)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: INVALID\n", 1)
	if err := os.WriteFile(fx.Issuer, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for invalid eligible EPC QR data")
	}
	if !strings.Contains(err.Error(), "issuer.payment.iban: invalid IBAN `INVALID`") {
		t.Fatalf("error %q does not contain the invalid IBAN message", err.Error())
	}
}

func TestRenderInvoiceRejectsNonSEPAEligibleEPCQRCode(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Issuer)
	if err != nil {
		t.Fatalf("ReadFile(issuerPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  iban: AT611904300234573201\n", "  iban: BR150000000000000000000000000\n", 1)
	if err := os.WriteFile(fx.Issuer, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(issuerPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte("\\usepackage{qrcode}\n@@EPC_QR_CODE@@\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	err = renderInvoice(t, h, templatePath, outputPath, ctx)
	if err == nil {
		t.Fatal("RenderInvoice returned nil error for non-SEPA eligible EPC QR data")
	}
	if !strings.Contains(err.Error(), "outside the current SEPA scheme scope") {
		t.Fatalf("error %q does not contain the non-SEPA IBAN message", err.Error())
	}
}

func TestStarterTemplateOmitsEPCSectionForNonEURInvoices(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Customers)
	if err != nil {
		t.Fatalf("ReadFile(customersPath) returned error: %v", err)
	}
	mutated := strings.TrimSpace(string(source)) + "\n  billing:\n    currency: USD\n"
	if err := os.WriteFile(fx.Customers, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customersPath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templateSource, err := os.ReadFile(filepath.Join("..", "..", "store", "starter", "template.tex"))
	if err != nil {
		t.Fatalf("ReadFile(starter/template.tex) returned error: %v", err)
	}
	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, templateSource, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
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
	t.Parallel()

	h := testfixture.NewHost(t)
	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0\n", "  paid_amount: 252\n", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	templateSource, err := os.ReadFile(filepath.Join("..", "..", "store", "starter", "template.tex"))
	if err != nil {
		t.Fatalf("ReadFile(starter/template.tex) returned error: %v", err)
	}
	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, templateSource, 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := renderInvoice(t, h, templatePath, outputPath, ctx); err != nil {
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
