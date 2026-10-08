package invoice

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/fsutil"
)

type Options struct {
	BaseDir           string
	CustomersPath     string
	IssuerPath        string
	DefaultsPath      string
	InvoicePath       string
	PDFPath           string
	TemplatePath      string
	OutputPath        string
	OverwriteOutput   bool
	EmailTo           string
	EmailSubject      string
	ArchiveAfterBuild bool
	AssumeYes         bool
	FromLastInvoice   bool
	EditNewInvoice    bool
}

// Context is an invoice with its customer and issuer, validated, with its
// totals computed.
type Context struct {
	CustomerID       string
	Customer         Customer
	Company          Company
	Payment          Payment
	Invoice          InvoiceHeader
	LineItems        []LineItem
	Currency         string
	VATBreakdowns    []VATBreakdown
	SubtotalCents    int64
	VATAmountCents   int64
	TotalCents       int64
	PaidAmountCents  int64
	OutstandingCents int64
	CustomerEmail    string
	InvoiceNumber    string
}

type LineItem struct {
	Name           string
	Description    string
	UnitPrice      *big.Rat
	Quantity       *big.Rat
	VATRatePercent *big.Rat
	LineTotalCents int64
}

type VATBreakdown struct {
	RatePercent    *big.Rat
	NetCents       int64
	VATAmountCents int64
}

const (
	configDirName                  = "invox"
	legacyConfigDirName            = "invoice-tool"
	tempBuildDirPrefix             = "invox-build-"
	epcQRAvailablePlaceholder      = "@@EPC_QR_AVAILABLE@@"
	epcQRLabelPlaceholder          = "@@EPC_QR_LABEL@@"
	epcQRCodePlaceholder           = "@@EPC_QR_CODE@@"
	lineItemsBeginPlaceholder      = "@@LINE_ITEMS_BEGIN@@"
	lineItemsEndPlaceholder        = "@@LINE_ITEMS_END@@"
	lineItemNamePlaceholder        = "@@LINE_ITEM_NAME@@"
	lineItemDescriptionPlaceholder = "@@LINE_ITEM_DESCRIPTION@@"
	lineItemUnitPricePlaceholder   = "@@LINE_ITEM_UNIT_PRICE@@"
	lineItemQuantityPlaceholder    = "@@LINE_ITEM_QUANTITY@@"
	lineItemVATRatePlaceholder     = "@@LINE_ITEM_VAT_RATE@@"
	lineItemLineTotalPlaceholder   = "@@LINE_ITEM_LINE_TOTAL@@"
	lineItemRulePlaceholder        = "@@LINE_ITEM_RULE@@"
	epcQRMaxPayloadBytes           = 331
	epcQRMaxNameChars              = 70
	epcQRMaxPurposeChars           = 4
	epcQRMaxTextChars              = 140
	epcQRMaxInfoChars              = 70
	epcQRMaxAmountCents            = 99999999999
)

var (
	epcPurposePattern        = regexp.MustCompile(`^[A-Za-z0-9]{1,4}$`)
	epcBICPattern            = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
	lineItemsBlockPattern    = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(lineItemsBeginPlaceholder) + `(.*?)` + regexp.QuoteMeta(lineItemsEndPlaceholder))
	lineItemsBoundaryPattern = regexp.MustCompile(regexp.QuoteMeta(lineItemsBeginPlaceholder) + `|` + regexp.QuoteMeta(lineItemsEndPlaceholder))
	ibanCountryLengths       = map[string]int{
		"AD": 24,
		"AE": 23,
		"AL": 28,
		"AT": 20,
		"AZ": 28,
		"BA": 20,
		"BE": 16,
		"BG": 22,
		"BH": 22,
		"BI": 16,
		"BR": 29,
		"BY": 28,
		"CH": 21,
		"CR": 22,
		"CY": 28,
		"CZ": 24,
		"DE": 22,
		"DJ": 27,
		"DK": 18,
		"DO": 28,
		"EE": 20,
		"EG": 29,
		"ES": 24,
		"FI": 18,
		"FK": 18,
		"FO": 18,
		"FR": 27,
		"GB": 22,
		"GE": 22,
		"GI": 23,
		"GL": 18,
		"GR": 27,
		"GT": 28,
		"HN": 28,
		"HR": 21,
		"HU": 28,
		"IE": 22,
		"IL": 23,
		"IQ": 23,
		"IS": 26,
		"IT": 27,
		"JO": 30,
		"KW": 30,
		"KZ": 20,
		"LB": 28,
		"LC": 32,
		"LI": 21,
		"LT": 20,
		"LU": 20,
		"LV": 21,
		"LY": 25,
		"MC": 27,
		"MD": 24,
		"ME": 22,
		"MK": 19,
		"MN": 20,
		"MR": 27,
		"MT": 31,
		"MU": 30,
		"NI": 32,
		"NL": 18,
		"NO": 15,
		"OM": 23,
		"PK": 24,
		"PL": 28,
		"PS": 29,
		"PT": 25,
		"QA": 29,
		"RO": 24,
		"RS": 22,
		"RU": 33,
		"SA": 24,
		"SC": 31,
		"SD": 18,
		"SE": 24,
		"SI": 19,
		"SK": 24,
		"SM": 27,
		"SO": 23,
		"ST": 25,
		"SV": 28,
		"TL": 23,
		"TN": 24,
		"TR": 26,
		"UA": 29,
		"VA": 22,
		"VG": 24,
		"XK": 20,
		"YE": 30,
	}
	// The EPC SEPA scope document lists both BIC and IBAN country codes.
	// This allowlist intentionally follows the IBAN code column because the
	// EPC QR validation is based on the beneficiary IBAN prefix. For example,
	// Guernsey, Jersey, and the Isle of Man are SEPA-reachable via the `GB`
	// IBAN prefix, while Gibraltar uses `GI`.
	sepaSchemeIBANCountryCodes = map[string]struct{}{
		"AD": {},
		"AL": {},
		"AT": {},
		"BE": {},
		"BG": {},
		"CH": {},
		"CY": {},
		"CZ": {},
		"DE": {},
		"DK": {},
		"EE": {},
		"ES": {},
		"FI": {},
		"FR": {},
		"GB": {},
		"GI": {},
		"GR": {},
		"HR": {},
		"HU": {},
		"IE": {},
		"IS": {},
		"IT": {},
		"LI": {},
		"LT": {},
		"LU": {},
		"LV": {},
		"MC": {},
		"MD": {},
		"ME": {},
		"MK": {},
		"MT": {},
		"NL": {},
		"NO": {},
		"PL": {},
		"PT": {},
		"RO": {},
		"RS": {},
		"SE": {},
		"SI": {},
		"SK": {},
		"SM": {},
		"VA": {},
	}
)

const defaultEPCQRLabel = "Pay via EPC-QR"

// NormalizeOptions makes every path in opts absolute, resolving relative
// paths against opts.BaseDir, the working directory the CLI was given.
func NormalizeOptions(opts *Options) {
	opts.BaseDir = filepath.Clean(opts.BaseDir)
	for _, path := range []*string{
		&opts.CustomersPath,
		&opts.IssuerPath,
		&opts.DefaultsPath,
		&opts.InvoicePath,
		&opts.PDFPath,
		&opts.TemplatePath,
		&opts.OutputPath,
	} {
		if *path != "" {
			*path = absPath(opts.BaseDir, *path)
		}
	}
}

// absPath is filepath.Abs with base in place of the process working
// directory.
func absPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if path != "" && os.IsPathSeparator(path[0]) {
		// A rooted path without a volume, such as \x on Windows, stays on
		// base's drive, as filepath.Abs keeps it on the current drive.
		return filepath.Join(filepath.VolumeName(base), path)
	}
	return filepath.Join(base, path)
}

func (h Host) GlobalCustomersPath() string {
	return filepath.Join(h.ConfigDir(), "customers.yaml")
}

func (h Host) GlobalIssuerPath() string {
	return filepath.Join(h.ConfigDir(), "issuer.yaml")
}

func (h Host) GlobalTemplatePath() string {
	return filepath.Join(h.ConfigDir(), "template.tex")
}

func (h Host) GlobalConfigPath() string {
	return filepath.Join(h.ConfigDir(), "config.yaml")
}

// EditableConfigPath returns the config file to open in an editor, creating
// it from the template when it does not exist: the explicit config file, else
// config.yaml where Config reads it, else config.yaml in the config
// directory.
func (h Host) EditableConfigPath() (string, error) {
	path := h.configFile
	if path == "" {
		found, err := h.findInConfigDir(false, "config.yaml")
		var missingDir *ConfigDirNotFoundError
		if err != nil && !errors.As(err, &missingDir) {
			return "", err
		}
		path = found.Path
	}
	if path == "" {
		if h.ConfigDir() == "" {
			return "", errors.New("config directory is unavailable")
		}
		path = h.GlobalConfigPath()
	}
	if err := fsutil.MkdirAll(filepath.Dir(path), fsutil.Private); err != nil {
		return "", err
	}
	if err := h.ensureConfigTemplate(path); err != nil {
		return "", err
	}
	return path, nil
}

func (h Host) ensureConfigTemplate(path string) error {
	_, err := ensureStarterFile(path, []byte(h.defaultConfigTemplate()), fsutil.Public)
	return err
}

func (h Host) defaultConfigTemplate() string {
	defaultArchiveDir := h.configTemplatePath(h.DefaultArchiveDir())
	if strings.TrimSpace(defaultArchiveDir) == "" {
		defaultArchiveDir = "invoices"
	}

	return strings.TrimLeft(fmt.Sprintf(`
# Invox user configuration.
#
# Uncomment a setting and change it to override the default.
#
# Supported settings:
#   paths.customers
#   paths.issuer
#   paths.defaults
#   paths.template
#   numbering.pattern
#   numbering.start
#   archive.dir
#     Directory where archived invoice files are stored.
#   email.subject
#     Subject template for the email command.
#   email.body
#     Plain-text body template for the email command.
#     Supported placeholders for email.subject and email.body:
#       {customer_name}
#       {email_greeting}
#       {contact_person}
#       {customer_id}
#       {invoice_number}
#       {issue_date}
#       {due_date}
#       {total_amount}
#       {outstanding_amount}
#       {payment_terms_text}
#       {issuer_name}
#
# Notes:
# - Top-level keys must not be indented.
# - Relative paths are resolved relative to this file.
# - "~/" expands to your home directory.
# - Per-customer numbering overrides live in customers.yaml at:
#   <customer>.numbering.start
# - Support file resolution order is:
#   1. explicit CLI flag
#   2. upward project search
#   3. paths.* in this file
#   4. conventional files in this config directory
#
# paths:
#   customers: 'customers.yaml'
#   issuer: 'issuer.yaml'
#   defaults: 'invoice_defaults.yaml'
#   template: 'template.tex'
#
# numbering:
#   pattern: '{customer_code}-{counter:03}'
#   start: 1
#
# archive:
#   dir: '%s'
#
# email:
#   subject: 'Invoice {invoice_number}'
#   body: |
#     {email_greeting}
#     
#     Please find attached invoice {invoice_number}.
#     Issue date: {issue_date}
#     Due date: {due_date}
#     Outstanding amount: {outstanding_amount}
#     
#     Regards,
#     {issuer_name}
`, defaultArchiveDir), "\n")
}

func (h Host) ConfigTemplate() string {
	return h.defaultConfigTemplate()
}

func (h Host) configTemplatePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if strings.TrimSpace(h.home) != "" {
		if path == h.home {
			path = "~"
		} else if strings.HasPrefix(path, h.home+string(os.PathSeparator)) {
			path = "~" + string(os.PathSeparator) + strings.TrimPrefix(path, h.home+string(os.PathSeparator))
		}
	}

	return filepath.ToSlash(path)
}

func (h Host) DefaultArchiveDir() string {
	baseDir := h.dataBase
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, configDirName, "invoices")
}

// LoadContext decodes the three files, validates them together and computes
// the invoice totals. It reports every problem at once: values that do not
// decode (*DecodeError, with file and line), then an unknown customer_id
// and the fields that are missing or out of range. Validation skips the
// fields that did not decode.
func LoadContext(customersPath, issuerPath, invoicePath string) (*Context, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return nil, err
	}
	var issuer IssuerFile
	issuerErr := decodeYAMLFile(issuerPath, &issuer, true)
	if issuerErr != nil && !isDecodeError(issuerErr) {
		return nil, issuerErr
	}
	var invoiceFile InvoiceFile
	invoiceErr := decodeYAMLFile(invoicePath, &invoiceFile, true)
	if invoiceErr != nil && !isDecodeError(invoiceErr) {
		return nil, invoiceErr
	}

	var problems []string
	var unknownCustomer error
	customerID := invoiceFile.CustomerID.Trim()
	// customer stays nil when the invoice names no usable customer, so its
	// fields are not reported missing one by one.
	var customer *Customer
	var customerErr error
	if customerID == "" {
		if !within("customer_id", failedFields(invoiceErr)) {
			problems = append(problems, fmt.Sprintf("%s: missing `customer_id`", invoicePath))
		}
	} else if found, ok, err := customers.customer(customerID, true); !ok {
		unknownCustomer = &UnknownCustomerError{Path: invoicePath, CustomerID: customerID}
	} else {
		customer, customerErr = &found, err
	}
	decodeErr := errors.Join(customerErr, issuerErr, invoiceErr)
	failed := failedFields(decodeErr)

	header := invoiceFile.Invoice
	if header == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `invoice` mapping", invoicePath))
		header = &InvoiceHeader{}
	}
	company := issuer.Company
	if company == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `company` mapping", issuerPath))
		company = &Company{}
	}
	payment := issuer.Payment
	if payment == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `payment` mapping", issuerPath))
		payment = &Payment{}
	}
	if len(invoiceFile.Positions) == 0 && !within("positions", failed) {
		problems = append(problems, fmt.Sprintf("%s: `positions` must be a non-empty list", invoicePath))
	}
	var fieldProblems []string
	if customer != nil {
		fieldProblems = append(fieldProblems, customer.validate()...)
	}
	fieldProblems = append(fieldProblems, company.validate()...)
	fieldProblems = append(fieldProblems, header.validate()...)
	fieldProblems = append(fieldProblems, payment.validate()...)

	var customerVATRate Rate
	if customer != nil {
		customerVATRate = customer.Tax.DefaultVATRate
	}
	// A rate that did not decode may be the one that applies, so a missing
	// rate is not reported while one did not decode.
	vatUndecoded := within("invoice.vat_percent", failed) || within("customer.tax.default_vat_rate", failed)
	items := make([]LineItem, 0, len(invoiceFile.Positions))
	missingVATReported := false
	for index, position := range invoiceFile.Positions {
		fieldProblems = append(fieldProblems, position.validate(index+1)...)
		rate := firstRate(position.VATPercent, header.VATPercent, customerVATRate)
		positionVATUndecoded := within(fmt.Sprintf("positions[%d].vat_percent", index+1), failed)
		if rate == nil && !missingVATReported && !vatUndecoded && !positionVATUndecoded {
			fieldProblems = append(fieldProblems, "invoice.vat_percent: missing value")
			missingVATReported = true
		}
		items = append(items, LineItem{
			Name:           string(position.Name),
			Description:    string(position.Description),
			UnitPrice:      position.UnitPrice.Rat(),
			Quantity:       position.Quantity.Rat(),
			VATRatePercent: rate,
		})
	}
	// Every field problem starts with the path of its field.
	for _, problem := range fieldProblems {
		if path, _, _ := strings.Cut(problem, ":"); !within(path, failed) {
			problems = append(problems, problem)
		}
	}

	var validationErr error
	if len(problems) > 0 {
		validationErr = errors.New(strings.Join(problems, "\n"))
	}
	if err := errors.Join(unknownCustomer, decodeErr, validationErr); err != nil {
		return nil, err
	}

	ctx := &Context{
		CustomerID:    customerID,
		Customer:      *customer,
		Company:       *company,
		Payment:       *payment,
		Invoice:       *header,
		Currency:      customer.BillingCurrency(),
		CustomerEmail: customer.InvoiceEmail(),
		InvoiceNumber: header.Number.Trim(),
	}
	// Validation bounded paid_amount, so it converts.
	ctx.PaidAmountCents, _ = moneyCents(header.PaidAmount.Rat())
	if err := ctx.computeTotals(items); err != nil {
		return nil, err
	}
	return ctx, nil
}

func isDecodeError(err error) bool {
	var decodeErr *DecodeError
	return errors.As(err, &decodeErr)
}

// firstRate returns the first rate that is set, nil when none is: the
// position's own, then the invoice's, then the customer's default.
func firstRate(rates ...Rate) *big.Rat {
	for _, rate := range rates {
		if rate.isSet() {
			return rate.Percent()
		}
	}
	return nil
}

// computeTotals sets the line totals, the VAT per rate and the invoice
// totals from items, whose prices, quantities and rates are validated.
func (ctx *Context) computeTotals(items []LineItem) error {
	var subtotalCents int64
	vatBuckets := make(map[string]*VATBreakdown, len(items))
	for index := range items {
		item := &items[index]
		lineTotal, ok := moneyCents(new(big.Rat).Mul(item.UnitPrice, item.Quantity))
		if !ok {
			return errAmountTooLarge(fmt.Sprintf("positions[%d]: unit_price × quantity", index+1))
		}
		if subtotalCents, ok = addMoneyCents(subtotalCents, lineTotal); !ok {
			return errAmountTooLarge("invoice subtotal")
		}
		item.LineTotalCents = lineTotal
		key := item.VATRatePercent.RatString()
		bucket, ok := vatBuckets[key]
		if !ok {
			bucket = &VATBreakdown{
				RatePercent: new(big.Rat).Set(item.VATRatePercent),
			}
			vatBuckets[key] = bucket
		}
		bucket.NetCents += lineTotal // bounded by the subtotal check above
	}

	vatBreakdowns := make([]VATBreakdown, 0, len(vatBuckets))
	var vatAmountCents int64
	for _, bucket := range vatBuckets {
		vatCents, ok := moneyCents(percentOfMoney(bucket.NetCents, bucket.RatePercent))
		if !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		bucket.VATAmountCents = vatCents
		if vatAmountCents, ok = addMoneyCents(vatAmountCents, vatCents); !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		vatBreakdowns = append(vatBreakdowns, *bucket)
	}
	sort.Slice(vatBreakdowns, func(left, right int) bool {
		return vatBreakdowns[left].RatePercent.Cmp(vatBreakdowns[right].RatePercent) < 0
	})

	totalCents, ok := addMoneyCents(subtotalCents, vatAmountCents)
	if !ok {
		return errAmountTooLarge("invoice total")
	}
	if ctx.PaidAmountCents > totalCents {
		return fmt.Errorf("invoice.paid_amount: `%s` exceeds total `%s`", FormatMoneyCents(ctx.PaidAmountCents), FormatMoneyCents(totalCents))
	}

	ctx.LineItems = items
	ctx.VATBreakdowns = vatBreakdowns
	ctx.SubtotalCents = subtotalCents
	ctx.VATAmountCents = vatAmountCents
	ctx.TotalCents = totalCents
	ctx.OutstandingCents = totalCents - ctx.PaidAmountCents
	return nil
}

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

func FormatCurrency(cents int64, currency string) string {
	return withCurrency(FormatMoneyCents(cents), currency)
}

// Unit prices are shown with as many decimals as they need, at least
// minUnitPriceDecimals and at most maxUnitPriceDecimals (rounded half up beyond
// that), so that unit price × quantity matches the line total.
const (
	minUnitPriceDecimals = 2
	maxUnitPriceDecimals = 4
)

func formatUnitPrice(value *big.Rat, currency string) string {
	decimals := unitPriceDecimals(value)
	if decimals == minUnitPriceDecimals {
		return FormatCurrency(quantizeMoney(value), currency)
	}
	scale := int64(1)
	for range decimals {
		scale *= 10
	}
	units := roundHalfUpToInt(new(big.Rat).Mul(value, new(big.Rat).SetInt64(scale)))
	sign := ""
	if units < 0 {
		sign = "-"
		units = -units
	}
	formatted := fmt.Sprintf("%s%s,%0*d", sign, groupThousands(units/scale), decimals, units%scale)
	return withCurrency(formatted, currency)
}

func unitPriceDecimals(value *big.Rat) int {
	if value == nil {
		return minUnitPriceDecimals
	}
	scaled := new(big.Rat).Set(value)
	scaled.Mul(scaled, big.NewRat(100, 1))
	for decimals := minUnitPriceDecimals; decimals < maxUnitPriceDecimals; decimals++ {
		if scaled.IsInt() {
			return decimals
		}
		scaled.Mul(scaled, big.NewRat(10, 1))
	}
	return maxUnitPriceDecimals
}

func withCurrency(formatted, currency string) string {
	if currency == "EUR" {
		return formatted + " \\euro"
	}
	return formatted + " " + latexEscape(currency)
}

func DisplayPath(path, baseDir string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func (h Host) expandHomePath(path string) string {
	return config.ExpandHome(path, h.home)
}

func PDFPathForOutput(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if ext == "" {
		return outputPath + ".pdf"
	}
	return strings.TrimSuffix(outputPath, ext) + ".pdf"
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

func validateLineItemBlockPlaceholders(template string, validationErrors *[]string) bool {
	depth := 0
	for _, match := range lineItemsBoundaryPattern.FindAllStringIndex(template, -1) {
		placeholder := template[match[0]:match[1]]
		switch placeholder {
		case lineItemsBeginPlaceholder:
			if depth > 0 {
				*validationErrors = append(*validationErrors, lineItemsBeginPlaceholder+": nested line-item blocks are unsupported")
				return false
			}
			depth++
		case lineItemsEndPlaceholder:
			if depth == 0 {
				*validationErrors = append(*validationErrors, lineItemsEndPlaceholder+": missing matching "+lineItemsBeginPlaceholder)
				return false
			}
			depth--
		}
	}
	if depth > 0 {
		*validationErrors = append(*validationErrors, lineItemsBeginPlaceholder+": missing matching "+lineItemsEndPlaceholder)
		return false
	}
	return true
}

func validateLineItemPlaceholdersOutsideBlocks(template string, validationErrors *[]string) {
	stripped := lineItemsBlockPattern.ReplaceAllString(template, "")
	for _, placeholder := range []string{
		lineItemNamePlaceholder,
		lineItemDescriptionPlaceholder,
		lineItemUnitPricePlaceholder,
		lineItemQuantityPlaceholder,
		lineItemVATRatePlaceholder,
		lineItemLineTotalPlaceholder,
		lineItemRulePlaceholder,
	} {
		if strings.Contains(stripped, placeholder) {
			*validationErrors = append(*validationErrors, fmt.Sprintf("%s: only supported inside %s ... %s", placeholder, lineItemsBeginPlaceholder, lineItemsEndPlaceholder))
		}
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

// renderLineItemTemplateBlocks substitutes every placeholder in a single pass:
// text outside line item blocks uses the template pairs, and each block body
// uses the line item pairs plus the template pairs. Substituted values are
// never scanned again.
func renderLineItemTemplateBlocks(template string, items []LineItem, currency string, templatePairs []string) string {
	replacer := strings.NewReplacer(templatePairs...)
	matches := lineItemsBlockPattern.FindAllStringSubmatchIndex(template, -1)
	if len(matches) == 0 {
		return replacer.Replace(template)
	}
	var builder strings.Builder
	lastEnd := 0
	for _, match := range matches {
		bounds := lineItemTemplateBlockBounds(template, match)
		builder.WriteString(replacer.Replace(template[lastEnd:bounds.renderStart]))
		body := template[bounds.bodyStart:bounds.bodyEnd]
		builder.WriteString(renderLineItemTemplateBlock(body, items, currency, templatePairs))
		lastEnd = bounds.renderEnd
	}
	builder.WriteString(replacer.Replace(template[lastEnd:]))
	return builder.String()
}

type lineItemBlockBounds struct {
	renderStart int
	bodyStart   int
	bodyEnd     int
	renderEnd   int
}

func lineItemTemplateBlockBounds(template string, match []int) lineItemBlockBounds {
	bounds := lineItemBlockBounds{
		renderStart: match[0],
		bodyStart:   match[2],
		bodyEnd:     match[3],
		renderEnd:   match[1],
	}

	if lineStart, lineAfter, ok := standaloneTemplateLineBounds(template, match[0], match[2]); ok {
		bounds.renderStart = lineStart
		bounds.bodyStart = lineAfter
	}
	if lineStart, lineAfter, ok := standaloneTemplateLineBounds(template, match[3], match[1]); ok {
		bounds.bodyEnd = lineStart
		bounds.renderEnd = lineAfter
	}

	return bounds
}

func standaloneTemplateLineBounds(template string, placeholderStart, placeholderEnd int) (int, int, bool) {
	lineStart := templateLineStart(template, placeholderStart)
	if !templateLineHasOnlyIndentation(template[lineStart:placeholderStart]) {
		return 0, 0, false
	}

	lineEnd := templateLineEnd(template, placeholderEnd)
	if !templateLineHasOnlyIndentation(template[placeholderEnd:lineEnd]) {
		return 0, 0, false
	}

	return lineStart, templateLineAfterBreak(template, lineEnd), true
}

func templateLineStart(text string, index int) int {
	for index > 0 {
		switch text[index-1] {
		case '\n', '\r':
			return index
		default:
			index--
		}
	}
	return 0
}

func templateLineEnd(text string, index int) int {
	for index < len(text) {
		switch text[index] {
		case '\n', '\r':
			return index
		default:
			index++
		}
	}
	return len(text)
}

func templateLineAfterBreak(text string, lineEnd int) int {
	if lineEnd >= len(text) {
		return lineEnd
	}
	if text[lineEnd] == '\r' && lineEnd+1 < len(text) && text[lineEnd+1] == '\n' {
		return lineEnd + 2
	}
	return lineEnd + 1
}

func templateLineHasOnlyIndentation(text string) bool {
	for _, r := range text {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

func renderLineItemTemplateBlock(body string, items []LineItem, currency string, templatePairs []string) string {
	var builder strings.Builder
	lastIndex := len(items) - 1
	for index, item := range items {
		builder.WriteString(renderLineItemTemplate(body, item, currency, lineItemRule(index, lastIndex), templatePairs))
	}
	return builder.String()
}

func renderLineItemTemplate(body string, item LineItem, currency, rule string, templatePairs []string) string {
	pairs := []string{
		lineItemNamePlaceholder, latexEscape(item.Name),
		lineItemDescriptionPlaceholder, latexEscape(item.Description),
		lineItemUnitPricePlaceholder, formatUnitPrice(item.UnitPrice, currency),
		lineItemQuantityPlaceholder, latexEscape(formatQuantity(item.Quantity)),
		lineItemVATRatePlaceholder, formatVATRate(item.VATRatePercent),
		lineItemLineTotalPlaceholder, FormatCurrency(item.LineTotalCents, currency),
		lineItemRulePlaceholder, rule,
	}
	return strings.NewReplacer(append(pairs, templatePairs...)...).Replace(body)
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

func epcQRCodeEligible(ctx *Context) bool {
	return ctx != nil && ctx.OutstandingCents > 0 && strings.TrimSpace(ctx.Currency) == "EUR"
}

func buildEPCPayload(ctx *Context) ([]byte, error) {
	if strings.TrimSpace(ctx.Currency) != "EUR" {
		return nil, fmt.Errorf("EPC QR code requires billing.currency EUR, got `%s`", ctx.Currency)
	}
	// EPC amounts run from 0.01 to 999999999.99.
	if ctx.OutstandingCents <= 0 {
		return nil, errors.New("invoice.outstanding_amount: EPC QR code requires an amount above zero")
	}
	if ctx.OutstandingCents > ctx.TotalCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds total `%s`", FormatMoneyCents(ctx.OutstandingCents), FormatMoneyCents(ctx.TotalCents))
	}
	if ctx.OutstandingCents > epcQRMaxAmountCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds EPC QR maximum `%s`", FormatMoneyCents(ctx.OutstandingCents), "999999999,99")
	}

	name := ctx.Payment.EPCQR.Name.Trim()
	if name == "" {
		name = ctx.Company.LegalCompanyName.Trim()
	}
	if name == "" {
		return nil, errors.New("issuer.payment.epc_qr.name: missing value")
	}

	iban := compactEPCAccountIdentifier(string(ctx.Payment.IBAN))
	if !isValidIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: invalid IBAN `%s`", ctx.Payment.IBAN)
	}
	if !isSEPASchemeIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: IBAN `%s` is outside the current SEPA scheme scope", ctx.Payment.IBAN)
	}

	bic := compactEPCAccountIdentifier(string(ctx.Payment.BIC))
	if bic != "" && !epcBICPattern.MatchString(bic) {
		return nil, fmt.Errorf("issuer.payment.bic: invalid BIC `%s`", ctx.Payment.BIC)
	}

	purpose := strings.ToUpper(ctx.Payment.EPCQR.Purpose.Trim())
	if purpose != "" && !epcPurposePattern.MatchString(purpose) {
		return nil, fmt.Errorf("issuer.payment.epc_qr.purpose: expected 1-4 letters or digits, got `%s`", purpose)
	}

	text := ctx.Payment.EPCQR.Text.Trim()
	if text == "" {
		text = ctx.Invoice.Number.Trim()
	}
	information := ctx.Payment.EPCQR.Information.Trim()

	for _, field := range []struct {
		label    string
		value    string
		maxChars int
	}{
		{label: "issuer.payment.epc_qr.name", value: name, maxChars: epcQRMaxNameChars},
		{label: "issuer.payment.epc_qr.text", value: text, maxChars: epcQRMaxTextChars},
		{label: "issuer.payment.epc_qr.information", value: information, maxChars: epcQRMaxInfoChars},
	} {
		if err := validateEPCTextField(field.label, field.value, field.maxChars); err != nil {
			return nil, err
		}
	}

	amount := "EUR" + formatEPCAmount(ctx.OutstandingCents)
	fields := []string{
		"BCD",
		"002",
		"1",
		"SCT",
		bic,
		name,
		iban,
		amount,
		purpose,
		"",
		text,
		information,
	}
	for len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	payload := strings.Join(fields, "\n")
	payloadBytes := []byte(payload)
	if len(payloadBytes) > epcQRMaxPayloadBytes {
		return nil, fmt.Errorf("EPC QR code payload exceeds %d bytes", epcQRMaxPayloadBytes)
	}
	return payloadBytes, nil
}

func validateEPCTextField(label, value string, maxChars int) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s: must be valid UTF-8", label)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s: line breaks are not allowed", label)
	}
	if utf8.RuneCountInString(value) > maxChars {
		return fmt.Errorf("%s: exceeds %d characters", label, maxChars)
	}
	return nil
}

func compactEPCAccountIdentifier(value string) string {
	value = strings.ToUpper(value)
	var compact strings.Builder
	compact.Grow(len(value))
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		compact.WriteRune(r)
	}
	return compact.String()
}

func isValidIBAN(value string) bool {
	if len(value) < 15 || len(value) > 34 {
		return false
	}
	countryCode := value[:2]
	expectedLength, ok := ibanCountryLengths[countryCode]
	if !ok || len(value) != expectedLength {
		return false
	}
	if value[2] < '0' || value[2] > '9' || value[3] < '0' || value[3] > '9' {
		return false
	}
	// ISO 13616 check digits are 98 minus a mod-97 remainder, so 00, 01 and
	// 99 never occur in a valid IBAN even when the checksum works out.
	if checkDigits := value[2:4]; checkDigits < "02" || checkDigits > "98" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	rearranged := value[4:] + value[:4]
	remainder := 0
	for _, r := range rearranged {
		switch {
		case r >= '0' && r <= '9':
			remainder = (remainder*10 + int(r-'0')) % 97
		case r >= 'A' && r <= 'Z':
			digits := int(r-'A') + 10
			remainder = (remainder*10 + digits/10) % 97
			remainder = (remainder*10 + digits%10) % 97
		default:
			return false
		}
	}
	return remainder == 1
}

func isSEPASchemeIBAN(value string) bool {
	if len(value) < 2 {
		return false
	}
	_, ok := sepaSchemeIBANCountryCodes[value[:2]]
	return ok
}

func formatEPCAmount(cents int64) string {
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func renderQRCodePayload(payload []byte) string {
	var rendered strings.Builder
	rendered.WriteString("{%\n")
	for value := 0x80; value <= 0xFF; value++ {
		fmt.Fprintf(&rendered, "\\catcode`\\^^%02x=12\\relax\n", value)
	}
	rendered.WriteString(`\edef\invoxqrcodepayload{`)
	rendered.WriteString(qrcodePayloadTeXSource(payload))
	rendered.WriteString("}%\n")
	rendered.WriteString(`\qrcode{\invoxqrcodepayload}`)
	rendered.WriteString("}")
	return rendered.String()
}

// qrcode parses a limited verbatim syntax in its argument. When the QR command
// is nested inside another macro, the package documentation requires spaces,
// reserved characters, and LF to reach \qrcode as escaped control sequences
// like \ , \%, \^, \~, \\, \{, \}, and \? rather than as raw TeX tokens.
func qrcodePayloadTeXSource(payload []byte) string {
	var source strings.Builder
	for _, value := range payload {
		switch value {
		case ' ':
			source.WriteString(`\noexpand\ `)
		case '\n':
			source.WriteString(`\noexpand\?`)
		case '\\':
			source.WriteString(`\noexpand\\`)
		case '%':
			source.WriteString(`\noexpand\%`)
		case '#':
			source.WriteString(`\noexpand\#`)
		case '&':
			source.WriteString(`\noexpand\&`)
		case '^':
			source.WriteString(`\noexpand\^`)
		case '_':
			source.WriteString(`\noexpand\_`)
		case '~':
			source.WriteString(`\noexpand\~`)
		case '$':
			source.WriteString(`\noexpand\$`)
		case '{':
			source.WriteString(`\noexpand\{`)
		case '}':
			source.WriteString(`\noexpand\}`)
		default:
			if value >= 0x80 {
				fmt.Fprintf(&source, "^^%02x", value)
				continue
			}
			source.WriteByte(value)
		}
	}
	return source.String()
}

func renderLineItems(items []LineItem, currency string) string {
	return renderLineItemRows(items, currency, false)
}

func renderLineItemsWithVAT(items []LineItem, currency string) string {
	return renderLineItemRows(items, currency, true)
}

func renderLineItemRows(items []LineItem, currency string, includeVAT bool) string {
	rows := make([]string, 0, len(items)*2)
	lastIndex := len(items) - 1
	for index, item := range items {
		parts := []string{
			latexEscape(item.Name),
			latexEscape(item.Description),
			formatUnitPrice(item.UnitPrice, currency),
			latexEscape(formatQuantity(item.Quantity)),
		}
		if includeVAT {
			parts = append(parts, formatVATRate(item.VATRatePercent))
		}
		parts = append(parts, FormatCurrency(item.LineTotalCents, currency))
		rows = append(rows, "    "+strings.Join(parts, " & ")+`\\`)
		rows = append(rows, "    "+lineItemRule(index, lastIndex))
	}
	return strings.Join(rows, "\n")
}

func lineItemRule(index, lastIndex int) string {
	ruleWidth := "0.2pt"
	if index == lastIndex {
		ruleWidth = "0.4pt"
	}
	return fmt.Sprintf(`\specialrule{%s}{0pt}{0pt}`, ruleWidth)
}

func renderVATSummaryRows(label string, breakdowns []VATBreakdown, currency string) string {
	rows := make([]string, 0, len(breakdowns))
	escapedLabel := latexEscape(label)
	for _, breakdown := range breakdowns {
		rows = append(rows, fmt.Sprintf(
			"%s (%s): & %s\\\\",
			escapedLabel,
			formatVATRate(breakdown.RatePercent),
			FormatCurrency(breakdown.VATAmountCents, currency),
		))
	}
	return strings.Join(rows, "\n")
}

func formatVATRate(value *big.Rat) string {
	return latexEscape(formatQuantity(value)) + `\%`
}

// decimalPattern is the grammar for money, quantities and rates: an optional
// minus sign, digits, and an optional fraction. Leading zeros are decimal.
var decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

func parseDecimal(text string) (*big.Rat, bool) {
	text = strings.TrimSpace(text)
	if !decimalPattern.MatchString(text) {
		return nil, false
	}
	rat := new(big.Rat)
	if _, ok := rat.SetString(text); ok {
		return rat, true
	}
	return nil, false
}

func percentOfMoney(cents int64, percent *big.Rat) *big.Rat {
	base := new(big.Rat).SetInt64(cents)
	base.Quo(base, big.NewRat(100, 1))
	result := new(big.Rat).Mul(base, percent)
	result.Quo(result, big.NewRat(100, 1))
	return result
}

func quantizeMoney(value *big.Rat) int64 {
	if value == nil {
		return 0
	}
	scaled := new(big.Rat).Mul(value, big.NewRat(100, 1))
	return roundHalfUpToInt(scaled)
}

func roundHalfUpToInt(value *big.Rat) int64 {
	if value == nil {
		return 0
	}
	return roundHalfUp(value).Int64()
}

// roundHalfUp rounds value to the nearest integer, with halves away from zero.
func roundHalfUp(value *big.Rat) *big.Int {
	numerator := new(big.Int).Set(value.Num())
	denominator := new(big.Int).Set(value.Denom())
	sign := numerator.Sign()
	if sign == 0 {
		return numerator
	}
	if sign < 0 {
		numerator.Neg(numerator)
	}
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	twiceRemainder := new(big.Int).Lsh(remainder, 1)
	if twiceRemainder.Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if sign < 0 {
		quotient.Neg(quotient)
	}
	return quotient
}

// FormatMoneyCents formats cents as 1.234,56, without a currency.
func FormatMoneyCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	integerPart := cents / 100
	fractionPart := cents % 100
	return fmt.Sprintf("%s%s,%02d", sign, groupThousands(integerPart), fractionPart)
}

func groupThousands(value int64) string {
	digits := fmt.Sprintf("%d", value)
	if len(digits) <= 3 {
		return digits
	}
	parts := make([]string, 0, (len(digits)+2)/3)
	for len(digits) > 3 {
		parts = append(parts, digits[len(digits)-3:])
		digits = digits[:len(digits)-3]
	}
	parts = append(parts, digits)
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, ".")
}

func formatQuantity(value *big.Rat) string {
	if value == nil {
		return ""
	}
	if value.Denom().Cmp(big.NewInt(1)) == 0 {
		return value.Num().String()
	}
	text := value.FloatString(10)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return strings.ReplaceAll(text, ".", ",")
}

func latexEscape(text string) string {
	replacer := strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`&`, `\&`,
		`%`, `\%`,
		`$`, `\$`,
		`#`, `\#`,
		`_`, `\_`,
		`{`, `\{`,
		`}`, `\}`,
		`~`, `\textasciitilde{}`,
		`^`, `\textasciicircum{}`,
	)
	escaped := replacer.Replace(text)
	// Values often follow \\ (line ends in addresses). LaTeX would read a
	// leading * as the starred form and a leading [ as an optional argument,
	// even after spaces, so an empty group shields them.
	if trimmed := strings.TrimLeft(escaped, " \t\r\n"); strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "*") {
		return "{}" + escaped
	}
	return escaped
}

func (h Host) copyTemplateAssets(templatePath, outputPath, rendered string) error {
	templateDir := filepath.Dir(templatePath)
	outputDir := filepath.Dir(outputPath)
	if templateDir == outputDir {
		return nil
	}

	for _, relDir := range referencedAssetDirs(rendered) {
		sourceDir := h.findAssetDir(templatePath, relDir)
		if sourceDir == "" {
			continue
		}
		destDir := filepath.Join(outputDir, relDir)
		if err := copyDir(sourceDir, destDir); err != nil {
			return err
		}
	}

	for _, relFile := range referencedAssetFiles(rendered) {
		sourceFile := h.findAssetFile(templatePath, relFile)
		if sourceFile == "" {
			continue
		}
		destFile := filepath.Join(outputDir, relFile)
		if fileExists(destFile) {
			continue
		}
		if err := copyFile(sourceFile, destFile); err != nil {
			return err
		}
	}

	return nil
}

func (h Host) findAssetDir(templatePath, relPath string) string {
	return h.findAsset(templatePath, relPath, true)
}

func (h Host) findAssetFile(templatePath, relPath string) string {
	return h.findAsset(templatePath, relPath, false)
}

// findAsset looks for relPath next to the template, then in the config
// directories.
func (h Host) findAsset(templatePath, relPath string, isDir bool) string {
	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(candidate, isDir) {
		return candidate
	}
	found, _ := h.findInConfigDir(isDir, relPath)
	return found.Path
}

func referencedAssetDirs(rendered string) []string {
	re := regexp.MustCompile(`Path=([^,\]\n]+)`)
	matches := re.FindAllStringSubmatch(rendered, -1)
	return uniqueRelativePaths(matches, 1)
}

func referencedAssetFiles(rendered string) []string {
	re := regexp.MustCompile(`\\includegraphics(?:\[[^\]]*\])?\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(rendered, -1)
	return uniqueRelativePaths(matches, 1)
}

func uniqueRelativePaths(matches [][]string, index int) []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) <= index {
			continue
		}
		path := strings.TrimSpace(match[index])
		if path == "" || filepath.IsAbs(path) || strings.HasPrefix(path, "..") {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
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

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if path != "" && fileExists(path) {
			return path
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
