package invoice

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
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
	EmailTo           string
	EmailSubject      string
	ArchiveAfterBuild bool
	FromLastInvoice   bool
}

type Context struct {
	CustomerID       string
	Customer         map[string]any
	IssuerCompany    map[string]any
	IssuerPayment    map[string]any
	Invoice          map[string]any
	LineItems        []LineItem
	Currency         string
	VATRatePercent   *big.Rat
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
	LineTotalCents int64
}

const (
	configDirName       = "invox"
	legacyConfigDirName = "invoice-tool"
	tempBuildDirPrefix  = "invox-build-"
)

func DefaultOptions() (Options, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Options{}, err
	}
	baseDir, err := filepath.Abs(cwd)
	if err != nil {
		return Options{}, err
	}
	customersPath, err := ResolveDefaultCustomersPath(cwd)
	if err != nil {
		return Options{}, err
	}
	issuerPath, err := ResolveDefaultIssuerPath(cwd)
	if err != nil {
		return Options{}, err
	}
	defaultsPath, err := ResolveDefaultInvoiceDefaultsPath(cwd)
	if err != nil {
		return Options{}, err
	}
	templatePath, err := ResolveDefaultTemplatePath(cwd)
	if err != nil {
		return Options{}, err
	}
	return Options{
		BaseDir:       baseDir,
		CustomersPath: customersPath,
		IssuerPath:    issuerPath,
		DefaultsPath:  defaultsPath,
		TemplatePath:  templatePath,
	}, nil
}

func NormalizeOptions(opts *Options) error {
	var err error
	opts.BaseDir, err = filepath.Abs(opts.BaseDir)
	if err != nil {
		return err
	}
	if opts.CustomersPath != "" {
		if opts.CustomersPath, err = filepath.Abs(opts.CustomersPath); err != nil {
			return err
		}
	}
	if opts.IssuerPath != "" {
		if opts.IssuerPath, err = filepath.Abs(opts.IssuerPath); err != nil {
			return err
		}
	}
	if opts.DefaultsPath != "" {
		if opts.DefaultsPath, err = filepath.Abs(opts.DefaultsPath); err != nil {
			return err
		}
	}
	if opts.InvoicePath != "" {
		if opts.InvoicePath, err = filepath.Abs(opts.InvoicePath); err != nil {
			return err
		}
	}
	if opts.PDFPath != "" {
		if opts.PDFPath, err = filepath.Abs(opts.PDFPath); err != nil {
			return err
		}
	}
	if opts.TemplatePath != "" {
		if opts.TemplatePath, err = filepath.Abs(opts.TemplatePath); err != nil {
			return err
		}
	}
	if opts.OutputPath != "" {
		if opts.OutputPath, err = filepath.Abs(opts.OutputPath); err != nil {
			return err
		}
	}
	return nil
}

func DiscoverBaseDir(start string) string {
	if path := findUpward(start, "invoice_template.tex"); path != "" {
		return filepath.Dir(path)
	}
	if path := findUpward(start, "customers.yaml"); path != "" {
		return filepath.Dir(path)
	}
	if path := findUpward(start, "issuer.yaml"); path != "" {
		return filepath.Dir(path)
	}
	if path := findUpward(start, "invoice_defaults.yaml"); path != "" {
		return filepath.Dir(path)
	}
	return start
}

func ConfigDir() string {
	baseDir := configHomeBaseDir()
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, configDirName)
}

func configHomeBaseDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return xdg
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".config")
}

func legacyConfigDir() string {
	baseDir := configHomeBaseDir()
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, legacyConfigDirName)
}

func configSearchDirs() []string {
	dirs := make([]string, 0, 2)
	if configDir := ConfigDir(); configDir != "" {
		dirs = append(dirs, configDir)
	}
	if legacyDir := legacyConfigDir(); legacyDir != "" && legacyDir != ConfigDir() {
		dirs = append(dirs, legacyDir)
	}
	return dirs
}

func configSearchPaths(names ...string) []string {
	paths := make([]string, 0, len(configSearchDirs())*len(names))
	for _, dir := range configSearchDirs() {
		for _, name := range names {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func GlobalCustomersPath() string {
	return filepath.Join(ConfigDir(), "customers.yaml")
}

func GlobalIssuerPath() string {
	return filepath.Join(ConfigDir(), "issuer.yaml")
}

func GlobalTemplatePath() string {
	return filepath.Join(ConfigDir(), "template.tex")
}

func GlobalConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func ResolveConfigPath() string {
	return firstExistingPath(configSearchPaths("config.yaml")...)
}

func EditableConfigPath() (string, error) {
	if path := ResolveConfigPath(); path != "" {
		if err := ensureConfigTemplate(path); err != nil {
			return "", err
		}
		return path, nil
	}

	path := GlobalConfigPath()
	if strings.TrimSpace(path) == "" {
		return "", errors.New("config directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := ensureConfigTemplate(path); err != nil {
		return "", err
	}
	return path, nil
}

func ensureConfigTemplate(path string) error {
	info, err := os.Stat(path)
	switch {
	case err == nil && info.Size() > 0:
		return nil
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return err
	}

	return os.WriteFile(path, []byte(defaultConfigTemplate()), 0o644)
}

func defaultConfigTemplate() string {
	defaultArchiveDir := configTemplatePath(DefaultArchiveDir())
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

func ConfigTemplate() string {
	return defaultConfigTemplate()
}

func configTemplatePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		if path == home {
			path = "~"
		} else if strings.HasPrefix(path, home+string(os.PathSeparator)) {
			path = "~" + string(os.PathSeparator) + strings.TrimPrefix(path, home+string(os.PathSeparator))
		}
	}

	return filepath.ToSlash(path)
}

func DefaultArchiveDir() string {
	baseDir := archiveDataBaseDir()
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, configDirName, "invoices")
}

func ResolveArchiveDir() (string, error) {
	path, err := resolveConfiguredPath("archive", "dir")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return DefaultArchiveDir(), nil
	}
	return path, nil
}

func ResolveDefaultCustomersPath(start string) (string, error) {
	return resolveDefaultPath(start, "customers", []string{"customers.yaml"}, []string{"customers.yaml"})
}

func ResolveDefaultIssuerPath(start string) (string, error) {
	return resolveDefaultPath(start, "issuer", []string{"issuer.yaml"}, []string{"issuer.yaml"})
}

func ResolveDefaultTemplatePath(start string) (string, error) {
	return resolveDefaultPath(start, "template", []string{"invoice_template.tex", "template.tex"}, []string{"template.tex", "invoice_template.tex"})
}

func resolveDefaultPath(start, configKey string, localNames, globalNames []string) (string, error) {
	if path := findUpward(start, localNames...); path != "" {
		return path, nil
	}

	path, err := resolveConfiguredPath("paths", configKey)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) != "" {
		return path, nil
	}

	return firstExistingPath(configSearchPaths(globalNames...)...), nil
}

func resolveConfiguredPath(section, key string) (string, error) {
	configPath, root, err := loadConfigRoot()
	if err != nil || strings.TrimSpace(configPath) == "" {
		return "", err
	}

	block, ok := root[section].(map[string]any)
	if !ok {
		return "", nil
	}

	rawPath := strings.TrimSpace(asString(block[key]))
	if rawPath == "" {
		return "", nil
	}

	return normalizeConfiguredPath(configPath, rawPath)
}

func resolveConfiguredString(section, key string) (string, error) {
	configPath, root, err := loadConfigRoot()
	if err != nil || strings.TrimSpace(configPath) == "" {
		return "", err
	}

	block, ok := root[section].(map[string]any)
	if !ok {
		return "", nil
	}

	rawValue, ok := block[key]
	if !ok || rawValue == nil {
		return "", nil
	}

	value, ok := rawValue.(string)
	if !ok {
		return "", fmt.Errorf("%s: %s.%s: expected a string", configPath, section, key)
	}
	return strings.TrimSpace(value), nil
}

func loadConfigRoot() (string, map[string]any, error) {
	configPath := ResolveConfigPath()
	if configPath == "" {
		return "", nil, nil
	}

	if err := validateConfigSource(configPath); err != nil {
		return "", nil, err
	}

	value, err := loadYAML(configPath)
	if err != nil {
		return "", nil, err
	}
	if value == nil {
		return configPath, map[string]any{}, nil
	}

	root, ok := value.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("%s: root value must be a mapping", configPath)
	}
	return configPath, root, nil
}

func validateConfigSource(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			return fmt.Errorf("%s: top-level keys must not be indented; remove the leading whitespace before %q", path, trimmed)
		}
		return nil
	}

	return nil
}

func LoadContext(customersPath, issuerPath, invoicePath string) (*Context, error) {
	customersValue, err := loadYAML(customersPath)
	if err != nil {
		return nil, err
	}
	issuerValue, err := loadYAML(issuerPath)
	if err != nil {
		return nil, err
	}
	invoiceValue, err := loadYAML(invoicePath)
	if err != nil {
		return nil, err
	}

	customers, ok := customersValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", customersPath)
	}
	issuerData, ok := issuerValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", issuerPath)
	}
	invoiceData, ok := invoiceValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", invoicePath)
	}

	var validationErrors []string

	customerID, _ := invoiceData["customer_id"].(string)
	customerID = strings.TrimSpace(customerID)
	var customer map[string]any
	if customerID == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: missing `customer_id`", invoicePath))
		customer = map[string]any{}
	} else {
		rawCustomer, ok := customers[customerID]
		if !ok {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: unknown customer_id `%s`", invoicePath, customerID))
			customer = map[string]any{}
		} else {
			customer, ok = rawCustomer.(map[string]any)
			if !ok {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: customer `%s` must be a mapping", customersPath, customerID))
				customer = map[string]any{}
			}
		}
	}

	invoiceBlock, ok := invoiceData["invoice"].(map[string]any)
	if !ok {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: missing `invoice` mapping", invoicePath))
		invoiceBlock = map[string]any{}
	}
	issuerCompany, ok := issuerData["company"].(map[string]any)
	if !ok {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: missing `company` mapping", issuerPath))
		issuerCompany = map[string]any{}
	}
	issuerPayment, ok := issuerData["payment"].(map[string]any)
	if !ok {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: missing `payment` mapping", issuerPath))
		issuerPayment = map[string]any{}
	}
	rawLineItems, ok := firstPresentPath(invoiceData, "positions", "line_items").([]any)
	if !ok || len(rawLineItems) == 0 {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: `positions` must be a non-empty list", invoicePath))
		rawLineItems = nil
	}

	if customerName(customer) == "" {
		validationErrors = append(validationErrors, "customer.name: missing value")
	}
	if customerEmail(customer) == "" {
		validationErrors = append(validationErrors, "customer.email: missing value")
	}
	requirePaths(customer, "customer", []string{
		"address.street",
		"address.postal_code",
		"address.city",
		"address.country",
		"tax.vat_tax_id",
	}, &validationErrors)
	requirePaths(issuerCompany, "issuer.company", []string{
		"legal_company_name",
		"company_registration_number",
		"vat_tax_id",
		"website",
		"email",
		"address.street",
		"address.postal_code",
		"address.city",
		"address.country",
	}, &validationErrors)
	requirePaths(invoiceBlock, "invoice", []string{
		"number",
		"issue_date",
		"due_date",
	}, &validationErrors)
	requireAnyPath(invoiceBlock, "invoice", "period", []string{"period", "period_label"}, &validationErrors)
	requirePaths(issuerPayment, "issuer.payment", []string{
		"bank_name",
		"iban",
		"bic",
		"due_days",
		"payment_terms_text",
	}, &validationErrors)

	for _, fieldName := range []string{"issue_date", "due_date"} {
		if value := getPath(invoiceBlock, fieldName); value != nil {
			if _, err := time.Parse("2006-01-02", fmt.Sprint(value)); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("invoice.%s: expected YYYY-MM-DD, got `%v`", fieldName, value))
			}
		}
	}

	paidAmount := coerceDecimal(getPath(invoiceBlock, "paid_amount"), "invoice.paid_amount", &validationErrors, true)
	vatRate := coerceVATRate(firstNonEmptyPath(invoiceBlock, "vat_percent", "vat_rate_percent"), customer, &validationErrors)
	coerceNonNegativeInt(getPath(issuerPayment, "due_days"), "issuer.payment.due_days", &validationErrors)

	normalizedItems := make([]LineItem, 0, len(rawLineItems))
	for index, rawItem := range rawLineItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			validationErrors = append(validationErrors, fmt.Sprintf("positions[%d]: each position must be a mapping", index+1))
			continue
		}
		requirePaths(item, fmt.Sprintf("positions[%d]", index+1), []string{"name", "description", "unit_price", "quantity"}, &validationErrors)
		unitPrice := coerceDecimal(item["unit_price"], fmt.Sprintf("positions[%d].unit_price", index+1), &validationErrors, false)
		quantity := coerceDecimal(item["quantity"], fmt.Sprintf("positions[%d].quantity", index+1), &validationErrors, false)
		if unitPrice != nil && unitPrice.Sign() < 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("positions[%d].unit_price: must be >= 0", index+1))
		}
		if quantity != nil && quantity.Sign() <= 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("positions[%d].quantity: must be > 0", index+1))
		}
		normalizedItems = append(normalizedItems, LineItem{
			Name:        asString(item["name"]),
			Description: asString(item["description"]),
			UnitPrice:   unitPrice,
			Quantity:    quantity,
		})
	}

	if len(validationErrors) > 0 {
		return nil, errors.New(strings.Join(validationErrors, "\n"))
	}

	var subtotalCents int64
	renderedItems := make([]LineItem, 0, len(normalizedItems))
	for _, item := range normalizedItems {
		lineTotal := quantizeMoney(new(big.Rat).Mul(item.UnitPrice, item.Quantity))
		subtotalCents += lineTotal
		item.LineTotalCents = lineTotal
		renderedItems = append(renderedItems, item)
	}

	vatAmountCents := quantizeMoney(percentOfMoney(subtotalCents, vatRate))
	totalCents := subtotalCents + vatAmountCents
	paidAmountCents := quantizeMoney(paidAmount)
	outstandingCents := totalCents - paidAmountCents

	if paidAmountCents > totalCents {
		return nil, fmt.Errorf("invoice.paid_amount: `%s` exceeds total `%s`", formatMoneyCents(paidAmountCents), formatMoneyCents(totalCents))
	}

	currency := customerCurrency(customer)
	resolvedCustomerEmail := customerEmail(customer)
	invoiceNumber := strings.TrimSpace(asString(getPath(invoiceBlock, "number")))
	normalizedInvoice := cloneMap(invoiceBlock)
	if period := firstNonEmptyPath(invoiceBlock, "period", "period_label"); period != nil {
		normalizedInvoice["period"] = period
	}
	if vatPercent := firstNonEmptyPath(invoiceBlock, "vat_percent", "vat_rate_percent"); vatPercent != nil {
		normalizedInvoice["vat_percent"] = vatPercent
	}

	return &Context{
		CustomerID:       customerID,
		Customer:         customer,
		IssuerCompany:    issuerCompany,
		IssuerPayment:    issuerPayment,
		Invoice:          normalizedInvoice,
		LineItems:        renderedItems,
		Currency:         currency,
		VATRatePercent:   vatRate,
		SubtotalCents:    subtotalCents,
		VATAmountCents:   vatAmountCents,
		TotalCents:       totalCents,
		PaidAmountCents:  paidAmountCents,
		OutstandingCents: outstandingCents,
		CustomerEmail:    resolvedCustomerEmail,
		InvoiceNumber:    invoiceNumber,
	}, nil
}

func RenderInvoice(templatePath, outputPath string, ctx *Context) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	rendered := string(content)
	for placeholder, value := range buildTemplateValues(ctx) {
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, []byte(rendered), 0o644); err != nil {
		return err
	}
	return copyTemplateAssets(templatePath, outputPath, rendered)
}

func BuildPDF(outputPath string) error {
	tectonicPath, err := exec.LookPath("tectonic")
	if err != nil {
		return errors.New("tectonic not found in PATH\nInstall it with `brew install tectonic`, then rerun this command.")
	}

	cmd := exec.Command(tectonicPath, filepath.Base(outputPath))
	cmd.Dir = filepath.Dir(outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func BuildInvoicePDF(templatePath, outputPath string, ctx *Context) error {
	tempDir, err := os.MkdirTemp("", tempBuildDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	renderPath := filepath.Join(
		tempDir,
		strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))+".tex",
	)
	if err := RenderInvoice(templatePath, renderPath, ctx); err != nil {
		return err
	}
	if err := BuildPDF(renderPath); err != nil {
		return err
	}
	return copyFile(PDFPathForOutput(renderPath), outputPath)
}

func FormatCurrency(cents int64, currency string) string {
	formatted := formatMoneyCents(cents)
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

func archiveDataBaseDir() string {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support")
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return appData
		}
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		return filepath.Join(home, "AppData", "Roaming")
	default:
		if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
			return xdg
		}
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		return filepath.Join(home, ".local", "share")
	}
}

func normalizeConfiguredPath(configPath, path string) (string, error) {
	resolved := expandHomePath(path)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(configPath), resolved)
	}
	return filepath.Abs(resolved)
}

func expandHomePath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return path
	}
	return filepath.Join(home, path[2:])
}

func PDFPathForOutput(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if ext == "" {
		return outputPath + ".pdf"
	}
	return strings.TrimSuffix(outputPath, ext) + ".pdf"
}

func buildTemplateValues(ctx *Context) map[string]string {
	return map[string]string{
		"@@ISSUER_NAME@@":                   latexEscape(asString(ctx.IssuerCompany["legal_company_name"])),
		"@@ISSUER_COMPANY_REG_NO@@":         latexEscape(asString(ctx.IssuerCompany["company_registration_number"])),
		"@@ISSUER_VAT_TAX_ID@@":             latexEscape(asString(ctx.IssuerCompany["vat_tax_id"])),
		"@@ISSUER_WEBSITE@@":                latexEscape(asString(ctx.IssuerCompany["website"])),
		"@@ISSUER_EMAIL@@":                  latexEscape(asString(ctx.IssuerCompany["email"])),
		"@@ISSUER_STREET@@":                 latexEscape(asString(getPath(ctx.IssuerCompany, "address.street"))),
		"@@ISSUER_CITY_AND_POSTAL_CODE@@":   latexEscape(fmt.Sprintf("%s %s", asString(getPath(ctx.IssuerCompany, "address.city")), asString(getPath(ctx.IssuerCompany, "address.postal_code")))),
		"@@ISSUER_COUNTRY@@":                latexEscape(asString(getPath(ctx.IssuerCompany, "address.country"))),
		"@@INVOICE_NUMBER@@":                latexEscape(asString(ctx.Invoice["number"])),
		"@@ISSUE_DATE@@":                    latexEscape(formatDate(asString(ctx.Invoice["issue_date"]))),
		"@@DUE_DATE@@":                      latexEscape(formatDate(asString(ctx.Invoice["due_date"]))),
		"@@INVOICE_TOTAL@@":                 FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@OUTSTANDING_TOTAL@@":             FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@CUSTOMER_NAME@@":                 latexEscape(customerName(ctx.Customer)),
		"@@CUSTOMER_STREET@@":               latexEscape(asString(getPath(ctx.Customer, "address.street"))),
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@": latexEscape(fmt.Sprintf("%s %s", asString(getPath(ctx.Customer, "address.city")), asString(getPath(ctx.Customer, "address.postal_code")))),
		"@@CUSTOMER_COUNTRY@@":              latexEscape(asString(getPath(ctx.Customer, "address.country"))),
		"@@CUSTOMER_VAT_TAX_ID@@":           latexEscape(asString(getPath(ctx.Customer, "tax.vat_tax_id"))),
		"@@CUSTOMER_EMAIL@@":                latexEscape(ctx.CustomerEmail),
		"@@LINE_ITEMS_ROWS@@":               renderLineItems(ctx.LineItems, ctx.Currency),
		"@@PERIOD_LABEL@@":                  latexEscape(asString(firstNonEmptyPath(ctx.Invoice, "period", "period_label"))),
		"@@PAYMENT_TERMS_TEXT@@":            latexEscape(asString(ctx.IssuerPayment["payment_terms_text"])),
		"@@SUBTOTAL@@":                      FormatCurrency(ctx.SubtotalCents, ctx.Currency),
		"@@VAT_RATE@@":                      latexEscape(formatQuantity(ctx.VATRatePercent)),
		"@@VAT_AMOUNT@@":                    FormatCurrency(ctx.VATAmountCents, ctx.Currency),
		"@@TOTAL@@":                         FormatCurrency(ctx.TotalCents, ctx.Currency),
		"@@PAID_AMOUNT@@":                   FormatCurrency(ctx.PaidAmountCents, ctx.Currency),
		"@@OUTSTANDING_AMOUNT@@":            FormatCurrency(ctx.OutstandingCents, ctx.Currency),
		"@@BANK_NAME@@":                     latexEscape(asString(ctx.IssuerPayment["bank_name"])),
		"@@IBAN@@":                          latexEscape(asString(ctx.IssuerPayment["iban"])),
		"@@BIC@@":                           latexEscape(asString(ctx.IssuerPayment["bic"])),
	}
}

func renderLineItems(items []LineItem, currency string) string {
	rows := make([]string, 0, len(items)*2)
	lastIndex := len(items) - 1
	for index, item := range items {
		rows = append(rows, fmt.Sprintf(
			"    %s & %s & %s & %s & %s\\\\",
			latexEscape(item.Name),
			latexEscape(item.Description),
			FormatCurrency(quantizeMoney(item.UnitPrice), currency),
			latexEscape(formatQuantity(item.Quantity)),
			FormatCurrency(item.LineTotalCents, currency),
		))
		ruleWidth := "0.2pt"
		if index == lastIndex {
			ruleWidth = "0.4pt"
		}
		rows = append(rows, fmt.Sprintf("    \\specialrule{%s}{0pt}{0pt}", ruleWidth))
	}
	return strings.Join(rows, "\n")
}

func requirePaths(source map[string]any, prefix string, paths []string, errors *[]string) {
	for _, path := range paths {
		value := getPath(source, path)
		if value == nil || strings.TrimSpace(asString(value)) == "" {
			*errors = append(*errors, fmt.Sprintf("%s.%s: missing value", prefix, path))
		}
	}
}

func requireAnyPath(source map[string]any, prefix, label string, paths []string, errors *[]string) {
	if firstNonEmptyPath(source, paths...) != nil {
		return
	}
	*errors = append(*errors, fmt.Sprintf("%s.%s: missing value", prefix, label))
}

func getPath(source any, path string) any {
	value := source
	for _, part := range strings.Split(path, ".") {
		mapping, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := mapping[part]
		if !ok {
			return nil
		}
		value = next
	}
	return value
}

func firstPresentPath(source map[string]any, paths ...string) any {
	for _, path := range paths {
		if value := getPath(source, path); value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyPath(source map[string]any, paths ...string) any {
	for _, path := range paths {
		value := getPath(source, path)
		if value != nil && strings.TrimSpace(asString(value)) != "" {
			return value
		}
	}
	return nil
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func coerceDecimal(value any, label string, errors *[]string, allowDefault bool) *big.Rat {
	if strings.TrimSpace(asString(value)) == "" {
		if allowDefault {
			return big.NewRat(0, 1)
		}
		*errors = append(*errors, fmt.Sprintf("%s: missing value", label))
		return nil
	}
	rat, ok := parseDecimal(asString(value))
	if !ok {
		*errors = append(*errors, fmt.Sprintf("%s: expected a number, got `%v`", label, value))
		return nil
	}
	return rat
}

func coerceVATRate(value any, customer map[string]any, errors *[]string) *big.Rat {
	raw := strings.TrimSpace(asString(value))
	if raw == "" {
		raw = strings.TrimSpace(asString(getPath(customer, "tax.default_vat_rate")))
	}
	if raw == "" {
		*errors = append(*errors, "invoice.vat_percent: missing value")
		return big.NewRat(0, 1)
	}
	raw = strings.TrimSuffix(raw, "%")
	rat, ok := parseDecimal(strings.TrimSpace(raw))
	if !ok {
		*errors = append(*errors, fmt.Sprintf("invoice.vat_percent: expected a number or percent string, got `%v`", value))
		return big.NewRat(0, 1)
	}
	return rat
}

func coerceNonNegativeInt(value any, label string, errors *[]string) int64 {
	raw := strings.TrimSpace(asString(value))
	if raw == "" {
		*errors = append(*errors, fmt.Sprintf("%s: missing value", label))
		return 0
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		*errors = append(*errors, fmt.Sprintf("%s: expected an integer, got `%v`", label, value))
		return 0
	}
	if parsed < 0 {
		*errors = append(*errors, fmt.Sprintf("%s: must be >= 0", label))
		return 0
	}
	return parsed
}

func parseDecimal(text string) (*big.Rat, bool) {
	rat := new(big.Rat)
	if _, ok := rat.SetString(strings.TrimSpace(text)); ok {
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
	numerator := new(big.Int).Set(value.Num())
	denominator := new(big.Int).Set(value.Denom())
	sign := numerator.Sign()
	if sign == 0 {
		return 0
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
	return quotient.Int64()
}

func formatDate(value string) string {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return t.Format("02.01.2006")
}

func formatMoneyCents(cents int64) string {
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
	return replacer.Replace(text)
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(value)
	}
}

func copyTemplateAssets(templatePath, outputPath, rendered string) error {
	templateDir := filepath.Dir(templatePath)
	outputDir := filepath.Dir(outputPath)
	if templateDir == outputDir {
		return nil
	}

	for _, relDir := range referencedAssetDirs(rendered) {
		sourceDir := findAssetDir(templatePath, relDir)
		if sourceDir == "" {
			continue
		}
		destDir := filepath.Join(outputDir, relDir)
		if err := copyDir(sourceDir, destDir); err != nil {
			return err
		}
	}

	for _, relFile := range referencedAssetFiles(rendered) {
		sourceFile := findAssetFile(templatePath, relFile)
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

func findAssetDir(templatePath, relPath string) string {
	for _, baseDir := range assetSearchDirs(templatePath) {
		candidate := filepath.Join(baseDir, relPath)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func findAssetFile(templatePath, relPath string) string {
	for _, baseDir := range assetSearchDirs(templatePath) {
		candidate := filepath.Join(baseDir, relPath)
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func assetSearchDirs(templatePath string) []string {
	dirs := []string{filepath.Dir(templatePath)}
	for _, configDir := range configSearchDirs() {
		if configDir != "" && configDir != dirs[0] {
			dirs = append(dirs, configDir)
		}
	}
	return dirs
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
			return os.MkdirAll(targetPath, info.Mode().Perm())
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
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, data, info.Mode().Perm())
}

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if path != "" && fileExists(path) {
			return path
		}
	}
	return ""
}

func prependPath(path string, paths []string) []string {
	return append([]string{path}, paths...)
}

func findUpward(start string, names ...string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if fileExists(candidate) {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
