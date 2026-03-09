package invoice

import (
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

func TestLoadContextSupportsLegacyInvoiceAliases(t *testing.T) {
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

	ctx, err := LoadContext(
		customersPath,
		issuerPath,
		legacyPath,
	)
	if err != nil {
		t.Fatalf("LoadContext returned error for legacy aliases: %v", err)
	}
	if got := asString(ctx.Invoice["period"]); got != "Leistungszeitraum" {
		t.Fatalf("Invoice[period] = %q, want %q", got, "Leistungszeitraum")
	}
	if got := asString(ctx.Invoice["vat_percent"]); got != "20" {
		t.Fatalf("Invoice[vat_percent] = %q, want %q", got, "20")
	}
	if len(ctx.LineItems) != 2 {
		t.Fatalf("len(LineItems) = %d, want 2", len(ctx.LineItems))
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

func TestCreateNewInvoiceRenamesLegacyDefaultKeys(t *testing.T) {
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
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	for _, want := range []string{"period:", "vat_percent: 20", "positions:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"period_label:", "vat_rate_percent:", "line_items:"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("created invoice still contains legacy key %q:\n%s", forbidden, text)
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
