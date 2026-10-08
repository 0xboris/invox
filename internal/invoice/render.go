package invoice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/render/latex"
)

const (
	tempBuildDirPrefix        = "invox-build-"
	epcQRAvailablePlaceholder = "@@EPC_QR_AVAILABLE@@"
	epcQRLabelPlaceholder     = "@@EPC_QR_LABEL@@"
	epcQRCodePlaceholder      = "@@EPC_QR_CODE@@"
)

const defaultEPCQRLabel = "Pay via EPC-QR"

func (h Host) RenderInvoice(templatePath, outputPath string, ctx *Context) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	template := latex.MigrateLegacyPlaceholders(string(content))
	if err := latex.ValidateTemplate(template); err != nil {
		return fmt.Errorf("%s: %w", templatePath, err)
	}
	hasActiveEPCQRAvailable := strings.Contains(template, epcQRAvailablePlaceholder)
	hasActiveEPCQRLabel := strings.Contains(template, epcQRLabelPlaceholder)
	hasActiveEPCQRCode := strings.Contains(template, epcQRCodePlaceholder)
	epcQRAvailable, epcQRLabel, epcQRCode, err := resolveEPCQRPlaceholders(
		ctx,
		hasActiveEPCQRAvailable,
		hasActiveEPCQRLabel,
		hasActiveEPCQRCode,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", templatePath, err)
	}
	values := buildTemplateValues(ctx)
	values[epcQRAvailablePlaceholder] = epcQRAvailable
	values[epcQRLabelPlaceholder] = epcQRLabel
	values[epcQRCodePlaceholder] = epcQRCode
	rendered := latex.Fill(template, values, latexItems(ctx.LineItems), ctx.Currency)
	if err := fsutil.WriteFile(outputPath, []byte(rendered), fsutil.Public); err != nil {
		return err
	}
	return h.copyTemplateAssets(templatePath, outputPath, rendered)
}

// BuildInvoicePDF renders the invoice into a temporary directory, runs
// compile on the .tex file there, and copies the PDF to outputPath.
func (h Host) BuildInvoicePDF(ctx context.Context, compile func(ctx context.Context, texPath string) error, templatePath, outputPath string, inv *Context) error {
	tempDir, err := os.MkdirTemp("", tempBuildDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	renderPath := filepath.Join(
		tempDir,
		strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))+".tex",
	)
	if err := h.RenderInvoice(templatePath, renderPath, inv); err != nil {
		return err
	}
	if err := compile(ctx, renderPath); err != nil {
		return err
	}
	pdf, err := os.ReadFile(PDFPathForOutput(renderPath))
	if err != nil {
		return err
	}
	return fsutil.WriteFile(outputPath, pdf, fsutil.Public)
}

func buildTemplateValues(ctx *Context) map[string]string {
	return map[string]string{
		"@@ISSUER_NAME@@":              latex.Escape(string(ctx.Company.LegalCompanyName)),
		"@@ISSUER_COMPANY_REG_NO@@":    latex.Escape(string(ctx.Company.CompanyRegistrationNumber)),
		"@@ISSUER_VAT_TAX_ID@@":        latex.Escape(string(ctx.Company.VATTaxID)),
		"@@ISSUER_WEBSITE@@":           latex.Escape(string(ctx.Company.Website)),
		"@@ISSUER_EMAIL@@":             latex.Escape(string(ctx.Company.Email)),
		"@@ISSUER_STREET@@":            latex.Escape(string(ctx.Company.Address.Street)),
		"@@ISSUER_CITY@@":              latex.Escape(string(ctx.Company.Address.City)),
		"@@ISSUER_POSTAL_CODE@@":       latex.Escape(string(ctx.Company.Address.PostalCode)),
		"@@ISSUER_COUNTRY@@":           latex.Escape(string(ctx.Company.Address.Country)),
		"@@INVOICE_NUMBER@@":           latex.Escape(string(ctx.Invoice.Number)),
		"@@ISSUE_DATE@@":               latex.Escape(ctx.Invoice.IssueDate.Display()),
		"@@DUE_DATE@@":                 latex.Escape(ctx.Invoice.DueDate.Display()),
		"@@INVOICE_TOTAL@@":            latex.FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@OUTSTANDING_TOTAL@@":        latex.FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@CUSTOMER_NAME@@":            latex.Escape(ctx.Customer.DisplayName()),
		"@@CUSTOMER_STREET@@":          latex.Escape(string(ctx.Customer.Address.Street)),
		"@@CUSTOMER_CITY@@":            latex.Escape(string(ctx.Customer.Address.City)),
		"@@CUSTOMER_POSTAL_CODE@@":     latex.Escape(string(ctx.Customer.Address.PostalCode)),
		"@@CUSTOMER_COUNTRY@@":         latex.Escape(string(ctx.Customer.Address.Country)),
		"@@CUSTOMER_VAT_TAX_ID@@":      latex.Escape(string(ctx.Customer.Tax.VATTaxID)),
		"@@CUSTOMER_EMAIL@@":           latex.Escape(ctx.CustomerEmail),
		"@@LINE_ITEMS_ROWS@@":          latex.LineItemRows(latexItems(ctx.LineItems), ctx.Currency),
		"@@LINE_ITEMS_ROWS_WITH_VAT@@": latex.LineItemRowsWithVAT(latexItems(ctx.LineItems), ctx.Currency),
		"@@PERIOD_LABEL@@":             latex.Escape(string(ctx.Invoice.Period)),
		"@@PAYMENT_TERMS_TEXT@@":       latex.Escape(string(ctx.Payment.PaymentTermsText)),
		"@@VAT_LABEL@@":                latex.Escape(ctx.Payment.vatLabel()),
		"@@SUBTOTAL@@":                 latex.FormatCurrency(ctx.SubtotalCents, ctx.Currency),
		"@@VAT_SUMMARY_ROWS@@":         latex.VATSummaryRows(ctx.Payment.vatLabel(), latexVATRows(ctx.VATBreakdowns), ctx.Currency),
		"@@TOTAL@@":                    latex.FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@PAID_AMOUNT@@":              latex.FormatCurrency(ctx.PaidAmountCents, ctx.Currency),
		"@@OUTSTANDING_AMOUNT@@":       latex.FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@BANK_NAME@@":                latex.Escape(string(ctx.Payment.BankName)),
		"@@IBAN@@":                     latex.Escape(string(ctx.Payment.IBAN)),
		"@@BIC@@":                      latex.Escape(string(ctx.Payment.BIC)),
	}
}

func latexItems(items []LineItem) []latex.Item {
	converted := make([]latex.Item, len(items))
	for i, item := range items {
		converted[i] = latex.Item(item)
	}
	return converted
}

func latexVATRows(breakdowns []VATBreakdown) []latex.VATRow {
	converted := make([]latex.VATRow, len(breakdowns))
	for i, breakdown := range breakdowns {
		converted[i] = latex.VATRow(breakdown)
	}
	return converted
}

func resolveEPCQRPlaceholders(ctx *Context, wantAvailable, wantLabel, wantCode bool) (string, string, string, error) {
	if !wantAvailable && !wantLabel && !wantCode {
		return "", "", "", nil
	}
	if !epcQRCodeEligible(ctx) {
		return epcQRAvailabilityLiteral(wantAvailable, false), "", "", nil
	}
	if !wantCode {
		return epcQRAvailabilityLiteral(wantAvailable, false), "", "", nil
	}

	payload, err := buildEPCPayload(ctx)
	if err != nil {
		return "", "", "", err
	}

	label := ""
	if wantLabel {
		label = renderEPCQRCodeLabel(ctx)
	}
	return epcQRAvailabilityLiteral(wantAvailable, true), label, latex.QRCode(payload), nil
}

func epcQRAvailabilityLiteral(wantAvailable, available bool) string {
	if !wantAvailable {
		return ""
	}
	if available {
		return "1"
	}
	return "0"
}

func renderEPCQRCodeLabel(ctx *Context) string {
	label := defaultEPCQRLabel
	if ctx != nil {
		if configured := ctx.Payment.EPCQR.Label.Trim(); configured != "" {
			label = configured
		}
	}
	return latex.Escape(label)
}
