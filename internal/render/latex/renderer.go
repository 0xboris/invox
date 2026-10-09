package latex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/invoice"
)

func buildTemplateValues(ctx *invoice.Context) map[string]string {
	return map[string]string{
		"@@ISSUER_NAME@@":              Escape(string(ctx.Company.LegalCompanyName)),
		"@@ISSUER_COMPANY_REG_NO@@":    Escape(string(ctx.Company.CompanyRegistrationNumber)),
		"@@ISSUER_VAT_TAX_ID@@":        Escape(string(ctx.Company.VATTaxID)),
		"@@ISSUER_WEBSITE@@":           Escape(string(ctx.Company.Website)),
		"@@ISSUER_EMAIL@@":             Escape(string(ctx.Company.Email)),
		"@@ISSUER_STREET@@":            Escape(string(ctx.Company.Address.Street)),
		"@@ISSUER_CITY@@":              Escape(string(ctx.Company.Address.City)),
		"@@ISSUER_POSTAL_CODE@@":       Escape(string(ctx.Company.Address.PostalCode)),
		"@@ISSUER_COUNTRY@@":           Escape(string(ctx.Company.Address.Country)),
		"@@INVOICE_NUMBER@@":           Escape(string(ctx.Header.Number)),
		"@@ISSUE_DATE@@":               Escape(ctx.Header.IssueDate.Display()),
		"@@DUE_DATE@@":                 Escape(ctx.Header.DueDate.Display()),
		"@@INVOICE_TOTAL@@":            FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@OUTSTANDING_TOTAL@@":        FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@CUSTOMER_NAME@@":            Escape(ctx.Customer.DisplayName()),
		"@@CUSTOMER_STREET@@":          Escape(string(ctx.Customer.Address.Street)),
		"@@CUSTOMER_CITY@@":            Escape(string(ctx.Customer.Address.City)),
		"@@CUSTOMER_POSTAL_CODE@@":     Escape(string(ctx.Customer.Address.PostalCode)),
		"@@CUSTOMER_COUNTRY@@":         Escape(string(ctx.Customer.Address.Country)),
		"@@CUSTOMER_VAT_TAX_ID@@":      Escape(string(ctx.Customer.Tax.VATTaxID)),
		"@@CUSTOMER_EMAIL@@":           Escape(ctx.CustomerEmail),
		"@@LINE_ITEMS_ROWS@@":          LineItemRows(latexItems(ctx.LineItems), ctx.Currency),
		"@@LINE_ITEMS_ROWS_WITH_VAT@@": LineItemRowsWithVAT(latexItems(ctx.LineItems), ctx.Currency),
		"@@PERIOD_LABEL@@":             Escape(string(ctx.Header.Period)),
		"@@PAYMENT_TERMS_TEXT@@":       Escape(string(ctx.Payment.PaymentTermsText)),
		"@@VAT_LABEL@@":                Escape(ctx.Payment.VATName()),
		"@@SUBTOTAL@@":                 FormatCurrency(ctx.SubtotalCents, ctx.Currency),
		"@@VAT_SUMMARY_ROWS@@":         VATSummaryRows(ctx.Payment.VATName(), latexVATRows(ctx.VATBreakdowns), ctx.Currency),
		"@@TOTAL@@":                    FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@PAID_AMOUNT@@":              FormatCurrency(ctx.PaidAmountCents, ctx.Currency),
		"@@OUTSTANDING_AMOUNT@@":       FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@BANK_NAME@@":                Escape(string(ctx.Payment.BankName)),
		"@@IBAN@@":                     Escape(string(ctx.Payment.IBAN)),
		"@@BIC@@":                      Escape(string(ctx.Payment.BIC)),
	}
}

func latexItems(items []invoice.LineItem) []Item {
	converted := make([]Item, len(items))
	for i, item := range items {
		converted[i] = Item(item)
	}
	return converted
}

func latexVATRows(breakdowns []invoice.VATBreakdown) []VATRow {
	converted := make([]VATRow, len(breakdowns))
	for i, breakdown := range breakdowns {
		converted[i] = VATRow(breakdown)
	}
	return converted
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

func copyDir(sourceDir, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, relPath)
		if info.IsDir() {
			return fsutil.MkdirAll(targetPath, fsutil.Perm{Dir: info.Mode().Perm()})
		}
		return copyFile(path, targetPath)
	})
}

func copyFile(sourcePath, destPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(destPath, data, fsutil.Perm{File: info.Mode().Perm(), Dir: 0o755})
}

const (
	epcQRAvailablePlaceholder = "@@EPC_QR_AVAILABLE@@"
	epcQRLabelPlaceholder     = "@@EPC_QR_LABEL@@"
	epcQRCodePlaceholder      = "@@EPC_QR_CODE@@"
)

// Renderer fills LaTeX templates in. It implements billing.Renderer.
type Renderer struct{}

var _ billing.Renderer = Renderer{}

// Render checks the template t and fills it in with ctx. It writes nothing.
func (Renderer) Render(t billing.Template, ctx *invoice.Context, epc billing.EPC) (string, error) {
	content, err := os.ReadFile(t.Path)
	if err != nil {
		return "", err
	}
	template := MigrateLegacyPlaceholders(string(content))
	if err := ValidateTemplate(template); err != nil {
		return "", fmt.Errorf("%s: %w", t.Path, err)
	}
	values := buildTemplateValues(ctx)
	wantAvailable := strings.Contains(template, epcQRAvailablePlaceholder)
	wantLabel := strings.Contains(template, epcQRLabelPlaceholder)
	wantCode := strings.Contains(template, epcQRCodePlaceholder)
	values[epcQRAvailablePlaceholder] = epcQRAvailabilityLiteral(wantAvailable, false)
	values[epcQRLabelPlaceholder] = ""
	values[epcQRCodePlaceholder] = ""
	// The code shows only where the template places it and the invoice is
	// paid by EPC transfer.
	if wantCode && (epc.Payload != nil || epc.Err != nil) {
		if epc.Err != nil {
			return "", fmt.Errorf("%s: %w", t.Path, epc.Err)
		}
		values[epcQRAvailablePlaceholder] = epcQRAvailabilityLiteral(wantAvailable, true)
		if wantLabel {
			values[epcQRLabelPlaceholder] = Escape(epc.Label)
		}
		values[epcQRCodePlaceholder] = QRCode(epc.Payload)
	}
	return Fill(template, values, latexItems(ctx.LineItems), ctx.Currency), nil
}

// Write writes source to path and copies the assets it uses from next to
// the template, or from the config directories, next to path.
func (Renderer) Write(t billing.Template, source, path string) error {
	if err := fsutil.WriteFile(path, []byte(source), fsutil.Public); err != nil {
		return err
	}
	outputDir := filepath.Dir(path)
	if filepath.Dir(t.Path) == outputDir {
		return nil
	}
	for _, relDir := range AssetDirs(source) {
		sourceDir := t.FindAsset(relDir, true)
		if sourceDir == "" {
			continue
		}
		if err := copyDir(sourceDir, filepath.Join(outputDir, relDir)); err != nil {
			return err
		}
	}
	for _, relFile := range AssetFiles(source) {
		sourceFile := t.FindAsset(relFile, false)
		if sourceFile == "" {
			continue
		}
		destFile := filepath.Join(outputDir, relFile)
		if info, err := os.Stat(destFile); err == nil && !info.IsDir() {
			continue
		}
		if err := copyFile(sourceFile, destFile); err != nil {
			return err
		}
	}
	return nil
}

// Build writes source with t's assets to a scratch directory, named after
// output, compiles it there with c, and copies the PDF to output.
func (r Renderer) Build(ctx context.Context, c billing.Compiler, t billing.Template, source, output string) error {
	dir, err := os.MkdirTemp("", "invox-build-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	sourcePath := filepath.Join(dir, strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".tex")
	if err := r.Write(t, source, sourcePath); err != nil {
		return err
	}
	pdf, err := c.Compile(ctx, sourcePath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(pdf)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(output, data, fsutil.Public)
}
