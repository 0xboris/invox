package invoice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xboris/invox/internal/fsutil"
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
	template := migrateLegacyTemplatePlaceholders(string(content))
	if err := validateTemplatePlaceholders(template); err != nil {
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
	rendered := renderLineItemTemplateBlocks(template, ctx.LineItems, ctx.Currency, sortedReplacementPairs(values))
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

func validateTemplatePlaceholders(template string) error {
	var validationErrors []string
	for placeholder, replacement := range map[string]string{
		"@@VAT_RATE@@":                      "@@VAT_SUMMARY_ROWS@@",
		"@@VAT_AMOUNT@@":                    "@@VAT_SUMMARY_ROWS@@",
		"@@ISSUER_CITY_AND_POSTAL_CODE@@":   "@@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@",
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@": "@@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@",
	} {
		if strings.Contains(template, placeholder) {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: unsupported placeholder; use %s", placeholder, replacement))
		}
	}
	if validateLineItemBlockPlaceholders(template, &validationErrors) {
		validateLineItemPlaceholdersOutsideBlocks(template, &validationErrors)
	}
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

func migrateLegacyTemplatePlaceholders(template string) string {
	legacyVATRowPattern := regexp.MustCompile(`(?m)^([ \t]*)VAT \(@@VAT_RATE@@\\%\): & @@VAT_AMOUNT@@\\\\[ \t]*$`)
	return legacyVATRowPattern.ReplaceAllString(template, `${1}@@VAT_SUMMARY_ROWS@@`)
}

func buildTemplateValues(ctx *Context) map[string]string {
	return map[string]string{
		"@@ISSUER_NAME@@":              latexEscape(string(ctx.Company.LegalCompanyName)),
		"@@ISSUER_COMPANY_REG_NO@@":    latexEscape(string(ctx.Company.CompanyRegistrationNumber)),
		"@@ISSUER_VAT_TAX_ID@@":        latexEscape(string(ctx.Company.VATTaxID)),
		"@@ISSUER_WEBSITE@@":           latexEscape(string(ctx.Company.Website)),
		"@@ISSUER_EMAIL@@":             latexEscape(string(ctx.Company.Email)),
		"@@ISSUER_STREET@@":            latexEscape(string(ctx.Company.Address.Street)),
		"@@ISSUER_CITY@@":              latexEscape(string(ctx.Company.Address.City)),
		"@@ISSUER_POSTAL_CODE@@":       latexEscape(string(ctx.Company.Address.PostalCode)),
		"@@ISSUER_COUNTRY@@":           latexEscape(string(ctx.Company.Address.Country)),
		"@@INVOICE_NUMBER@@":           latexEscape(string(ctx.Invoice.Number)),
		"@@ISSUE_DATE@@":               latexEscape(ctx.Invoice.IssueDate.Display()),
		"@@DUE_DATE@@":                 latexEscape(ctx.Invoice.DueDate.Display()),
		"@@INVOICE_TOTAL@@":            FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@OUTSTANDING_TOTAL@@":        FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@CUSTOMER_NAME@@":            latexEscape(ctx.Customer.DisplayName()),
		"@@CUSTOMER_STREET@@":          latexEscape(string(ctx.Customer.Address.Street)),
		"@@CUSTOMER_CITY@@":            latexEscape(string(ctx.Customer.Address.City)),
		"@@CUSTOMER_POSTAL_CODE@@":     latexEscape(string(ctx.Customer.Address.PostalCode)),
		"@@CUSTOMER_COUNTRY@@":         latexEscape(string(ctx.Customer.Address.Country)),
		"@@CUSTOMER_VAT_TAX_ID@@":      latexEscape(string(ctx.Customer.Tax.VATTaxID)),
		"@@CUSTOMER_EMAIL@@":           latexEscape(ctx.CustomerEmail),
		"@@LINE_ITEMS_ROWS@@":          renderLineItems(ctx.LineItems, ctx.Currency),
		"@@LINE_ITEMS_ROWS_WITH_VAT@@": renderLineItemsWithVAT(ctx.LineItems, ctx.Currency),
		"@@PERIOD_LABEL@@":             latexEscape(string(ctx.Invoice.Period)),
		"@@PAYMENT_TERMS_TEXT@@":       latexEscape(string(ctx.Payment.PaymentTermsText)),
		"@@VAT_LABEL@@":                latexEscape(ctx.Payment.vatLabel()),
		"@@SUBTOTAL@@":                 FormatCurrency(ctx.SubtotalCents, ctx.Currency),
		"@@VAT_SUMMARY_ROWS@@":         renderVATSummaryRows(ctx.Payment.vatLabel(), ctx.VATBreakdowns, ctx.Currency),
		"@@TOTAL@@":                    FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@PAID_AMOUNT@@":              FormatCurrency(ctx.PaidAmountCents, ctx.Currency),
		"@@OUTSTANDING_AMOUNT@@":       FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@BANK_NAME@@":                latexEscape(string(ctx.Payment.BankName)),
		"@@IBAN@@":                     latexEscape(string(ctx.Payment.IBAN)),
		"@@BIC@@":                      latexEscape(string(ctx.Payment.BIC)),
	}
}

// sortedReplacementPairs flattens placeholder values into strings.NewReplacer
// arguments in sorted key order, so rendering does not depend on map order.
func sortedReplacementPairs(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, key, values[key])
	}
	return pairs
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
	return epcQRAvailabilityLiteral(wantAvailable, true), label, renderQRCodePayload(payload), nil
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
	return latexEscape(label)
}
