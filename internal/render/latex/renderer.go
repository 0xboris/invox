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

// invoicePlaceholders are the template placeholders filled from the invoice
// alone, with their values. Placeholders lists them with the others.
var invoicePlaceholders = []struct {
	name  string
	value func(*invoice.Context) string
}{
	{"@@ISSUER_NAME@@", func(c *invoice.Context) string { return Escape(string(c.Company.LegalCompanyName)) }},
	{"@@ISSUER_COMPANY_REG_NO@@", func(c *invoice.Context) string { return Escape(string(c.Company.CompanyRegistrationNumber)) }},
	{"@@ISSUER_VAT_TAX_ID@@", func(c *invoice.Context) string { return Escape(string(c.Company.VATTaxID)) }},
	{"@@ISSUER_WEBSITE@@", func(c *invoice.Context) string { return Escape(string(c.Company.Website)) }},
	{"@@ISSUER_EMAIL@@", func(c *invoice.Context) string { return Escape(string(c.Company.Email)) }},
	{"@@ISSUER_STREET@@", func(c *invoice.Context) string { return Escape(string(c.Company.Address.Street)) }},
	{"@@ISSUER_CITY@@", func(c *invoice.Context) string { return Escape(string(c.Company.Address.City)) }},
	{"@@ISSUER_POSTAL_CODE@@", func(c *invoice.Context) string { return Escape(string(c.Company.Address.PostalCode)) }},
	{"@@ISSUER_COUNTRY@@", func(c *invoice.Context) string { return Escape(string(c.Company.Address.Country)) }},
	{"@@CUSTOMER_NAME@@", func(c *invoice.Context) string { return Escape(c.Customer.DisplayName()) }},
	{"@@CUSTOMER_STREET@@", func(c *invoice.Context) string { return Escape(string(c.Customer.Address.Street)) }},
	{"@@CUSTOMER_CITY@@", func(c *invoice.Context) string { return Escape(string(c.Customer.Address.City)) }},
	{"@@CUSTOMER_POSTAL_CODE@@", func(c *invoice.Context) string { return Escape(string(c.Customer.Address.PostalCode)) }},
	{"@@CUSTOMER_COUNTRY@@", func(c *invoice.Context) string { return Escape(string(c.Customer.Address.Country)) }},
	{"@@CUSTOMER_VAT_TAX_ID@@", func(c *invoice.Context) string { return Escape(string(c.Customer.Tax.VATTaxID)) }},
	{"@@CUSTOMER_EMAIL@@", func(c *invoice.Context) string { return Escape(c.CustomerEmail) }},
	{"@@INVOICE_NUMBER@@", func(c *invoice.Context) string { return Escape(string(c.Header.Number)) }},
	{"@@ISSUE_DATE@@", func(c *invoice.Context) string { return Escape(c.Header.IssueDate.Display()) }},
	{"@@DUE_DATE@@", func(c *invoice.Context) string { return Escape(c.Header.DueDate.Display()) }},
	{"@@PERIOD_LABEL@@", func(c *invoice.Context) string { return Escape(string(c.Header.Period)) }},
	{"@@LINE_ITEMS_ROWS@@", func(c *invoice.Context) string { return LineItemRows(c.LineItems, c.Currency) }},
	{"@@LINE_ITEMS_ROWS_WITH_VAT@@", func(c *invoice.Context) string { return LineItemRowsWithVAT(c.LineItems, c.Currency) }},
	{"@@SUBTOTAL@@", func(c *invoice.Context) string { return FormatCurrency(c.SubtotalCents, c.Currency) }},
	{"@@VAT_SUMMARY_ROWS@@", func(c *invoice.Context) string {
		return VATSummaryRows(c.Payment.VATName(), c.VATBreakdowns, c.Currency)
	}},
	{"@@TOTAL@@", func(c *invoice.Context) string { return FormatCurrency(c.TotalCents, c.Currency) }},
	{"@@PAID_AMOUNT@@", func(c *invoice.Context) string { return FormatCurrency(c.PaidAmountCents, c.Currency) }},
	{"@@OUTSTANDING_AMOUNT@@", func(c *invoice.Context) string { return FormatCurrency(c.OutstandingCents, c.Currency) }},
	{"@@INVOICE_TOTAL@@", func(c *invoice.Context) string { return FormatCurrency(c.TotalCents, c.Currency) }},
	{"@@OUTSTANDING_TOTAL@@", func(c *invoice.Context) string { return FormatCurrency(c.OutstandingCents, c.Currency) }},
	{"@@PAYMENT_TERMS_TEXT@@", func(c *invoice.Context) string { return Escape(string(c.Payment.PaymentTermsText)) }},
	{"@@VAT_LABEL@@", func(c *invoice.Context) string { return Escape(c.Payment.VATName()) }},
	{"@@BANK_NAME@@", func(c *invoice.Context) string { return Escape(string(c.Payment.BankName)) }},
	{"@@IBAN@@", func(c *invoice.Context) string { return Escape(string(c.Payment.IBAN)) }},
	{"@@BIC@@", func(c *invoice.Context) string { return Escape(string(c.Payment.BIC)) }},
}

func buildTemplateValues(ctx *invoice.Context) map[string]string {
	values := make(map[string]string, len(invoicePlaceholders))
	for _, p := range invoicePlaceholders {
		values[p.name] = p.value(ctx)
	}
	return values
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

// Compiler turns LaTeX source into a PDF and returns its path. The renderer
// is its only caller, so it is declared here rather than in billing.
type Compiler interface {
	Compile(ctx context.Context, sourcePath string) (string, error)
}

// Renderer fills LaTeX templates in. It implements billing.Renderer.
type Renderer struct {
	// FindAsset returns the file or directory rel that the template at
	// template uses: next to it, else in the config directory. It returns
	// "" when there is none.
	FindAsset func(template, rel string, dir bool) string
	// Compiler compiles what Build writes.
	Compiler Compiler
}

var _ billing.Renderer = Renderer{}

// Render checks the template t and fills it in with ctx. It writes nothing.
func (Renderer) Render(t billing.Template, ctx *invoice.Context, epc billing.EPC) (string, error) {
	content, err := os.ReadFile(t.Path)
	if err != nil {
		return "", err
	}
	template := string(content)
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
	return Fill(template, values, ctx.LineItems, ctx.Currency), nil
}

// Write writes source to path and copies the assets it uses from next to
// the template, or from the config directories, next to path. It returns a
// *billing.OutputIsDirError when path is a directory.
func (r Renderer) Write(t billing.Template, source, path string) error {
	if err := refuseDir(path); err != nil {
		return err
	}
	if err := fsutil.WriteFile(path, []byte(source), fsutil.Public); err != nil {
		return err
	}
	outputDir := filepath.Dir(path)
	if filepath.Dir(t.Path) == outputDir {
		return nil
	}
	for _, relDir := range AssetDirs(source) {
		sourceDir := r.FindAsset(t.Path, relDir, true)
		if sourceDir == "" {
			continue
		}
		if err := copyDir(sourceDir, filepath.Join(outputDir, relDir)); err != nil {
			return err
		}
	}
	for _, relFile := range AssetFiles(source) {
		sourceFile := r.FindAsset(t.Path, relFile, false)
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
// output, compiles it there with r.Compiler, and copies the PDF to output.
// It returns a *billing.OutputIsDirError, before compiling, when output is
// a directory.
func (r Renderer) Build(ctx context.Context, t billing.Template, source, output string) error {
	if err := refuseDir(output); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "invox-build-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	sourcePath := filepath.Join(dir, strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".tex")
	if err := r.Write(t, source, sourcePath); err != nil {
		return err
	}
	pdf, err := r.Compiler.Compile(ctx, sourcePath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(pdf)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(output, data, fsutil.Public)
}

// refuseDir returns a *billing.OutputIsDirError when path is a directory or
// a symlink to one.
func refuseDir(path string) error {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return &billing.OutputIsDirError{Path: path}
	}
	return nil
}
