package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRequiresCustomerID(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"new"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required arguments: CUSTOMER_ID") {
		t.Fatalf("stderr %q does not contain missing CUSTOMER_ID message", stderr)
	}
}

func TestCustomerHelpShowsCustomerSubcommands(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"customer", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox customer <subcommand> [options]",
		"list          List all customers",
		"config        Open customers.yaml in the default shell editor",
		"-c, --customers PATH",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestConfigOpensConfigFile(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	openedPath := ""
	oldOpenTextFile := openTextFile
	openTextFile = func(path string) error {
		openedPath = path
		return nil
	}
	t.Cleanup(func() {
		openTextFile = oldOpenTextFile
	})

	exitCode, stdout, stderr := captureRun(t, []string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	wantPath := filepath.Join(configHome, "invox", "config.yaml")
	if openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	source, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}
	for _, want := range []string{
		"# Invox user configuration.",
		"#   paths.customers",
		"#   paths.template",
		"#   numbering.pattern",
		"#   numbering.start",
		"#   archive.dir",
	} {
		if !strings.Contains(string(source), want) {
			t.Fatalf("config template %q does not contain %q", string(source), want)
		}
	}
	if !strings.Contains(stdout, "Opened "+wantPath) {
		t.Fatalf("stdout %q does not contain opened path", stdout)
	}
}

func TestConfigHelpShowsUsage(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"config", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Open config.yaml in the default shell editor.",
		"invox help config",
		"Formatting:",
		"Top-level keys must start at column 1 with no leading spaces.",
		"Supported settings:",
		"paths.customers",
		"paths.template",
		"numbering.pattern",
		"numbering.start",
		"archive.dir",
		"Customer overrides:",
		"customers.<CUSTOMER_ID>.numbering.start",
		"Support file precedence:",
		"Template:",
		"# archive:",
		"invox config",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestHelpConfigShowsConfigDocumentation(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"help", "config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Formatting:",
		"Top-level keys must start at column 1 with no leading spaces.",
		"Supported settings:",
		"paths.customers",
		"numbering.pattern",
		"customers.<CUSTOMER_ID>.numbering.start",
		"archive.dir",
		"Template:",
		"# paths:",
		"# archive:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestCustomerConfigOpensCustomersFile(t *testing.T) {
	customersPath := filepath.Join(t.TempDir(), "customers.yaml")
	if err := os.WriteFile(customersPath, []byte("CUST-001: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	openedPath := ""
	oldOpenTextFile := openTextFile
	openTextFile = func(path string) error {
		openedPath = path
		return nil
	}
	t.Cleanup(func() {
		openTextFile = oldOpenTextFile
	})

	exitCode, stdout, stderr := captureRun(t, []string{
		"customer",
		"config",
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if openedPath != customersPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, customersPath)
	}
	if !strings.Contains(stdout, "Opened "+customersPath) {
		t.Fatalf("stdout %q does not contain opened path", stdout)
	}
}

func TestCustomerConfigHelpShowsConfigUsage(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"customer", "config", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Open customers.yaml in the default shell editor.",
		"invox customer config [-c CUSTOMERS.yaml]",
		"-c, --customers PATH",
		"<customer>.name",
		"<customer>.email",
		"<customer>.billing.currency",
		"<customer>.numbering.start",
		"invox customer config",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestCustomerConfigReportsIndentedTopLevelConfig(t *testing.T) {
	writeConfigFile(t, " numbering:\n  pattern: '{customer_id}-{counter:03}'\npaths:\n  customers: '~/customers.yaml'\n")

	exitCode, stdout, stderr := captureRun(t, []string{"customer", "config"})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "top-level keys must not be indented") {
		t.Fatalf("stderr %q does not contain indentation error", stderr)
	}
}

func TestResolveShellEditorPrefersVisualThenEditor(t *testing.T) {
	t.Setenv("VISUAL", "hx")
	t.Setenv("EDITOR", "vim")
	if got := resolveShellEditor(); got != "hx" {
		t.Fatalf("resolveShellEditor() = %q, want %q", got, "hx")
	}

	t.Setenv("VISUAL", "")
	if got := resolveShellEditor(); got != "vim" {
		t.Fatalf("resolveShellEditor() = %q, want %q", got, "vim")
	}
}

func TestCustomerListPrintsCustomers(t *testing.T) {
	customersPath, _, _, _ := writeContextFixtures(t)
	exitCode, stdout, stderr := captureRun(t, []string{
		"customer",
		"list",
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "CUST-001\tAppsters GmbH\tactive") {
		t.Fatalf("stdout %q does not contain expected customer row", stdout)
	}
}

func TestCustomerListPreservesUnquotedNumericLookingCustomerID(t *testing.T) {
	customersPath := filepath.Join(t.TempDir(), "customers.yaml")
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
0021:
  status: active
  name: Appsters GmbH
  email: office@appsters.at
  address:
    street: Griesgasse 19
    postal_code: "9020"
    city: Klagenfurt
    country: Oesterreich
  tax:
    vat_tax_id: ATU80037005
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"customer",
		"list",
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "0021\tAppsters GmbH\tactive") {
		t.Fatalf("stdout %q does not contain preserved customer ID", stdout)
	}
}

func TestNewHelpShowsShortFlags(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"new", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"CUSTOMER_ID",
		"-c, --customers PATH",
		"-u, --issuer PATH",
		"-s, --source PATH",
		"-o, --output PATH",
		"invoice.yaml in the current directory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestNewCreatesDefaultOutputInvoiceFile(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	workDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	t.Chdir(workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"new",
		"CUST-001",
		"-c", customersPath,
		"-u", issuerPath,
		"-s", defaultsPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created invoice.yaml for CUST-001") {
		t.Fatalf("stdout %q does not contain success output", stdout)
	}
	if _, err := os.Stat(filepath.Join(workDir, "invoice.yaml")); err != nil {
		t.Fatalf("default invoice.yaml was not created: %v", err)
	}
}

func TestNewUsesCustomerSpecificStartFromCustomersFile(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixturesWithCustomerStart(t, "7")
	workDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	t.Chdir(workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"new",
		"CUST-001",
		"-c", customersPath,
		"-u", issuerPath,
		"-s", defaultsPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created invoice.yaml for CUST-001 (CUST-001-007)") {
		t.Fatalf("stdout %q does not contain customer-specific invoice number", stdout)
	}
}

func TestIncrementRequiresInput(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"increment"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required flags: -i, --input") {
		t.Fatalf("stderr %q does not contain missing input message", stderr)
	}
}

func TestRenderRequiresInputOnly(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"render"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required flags: -i, --input") {
		t.Fatalf("stderr %q does not contain missing flag message", stderr)
	}
	if !strings.Contains(stderr, "invox render -i INVOICE.yaml [-o OUTPUT.tex]") {
		t.Fatalf("stderr %q does not contain usage", stderr)
	}
}

func TestRenderHelpShowsShortFlags(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"render", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"-i, --input PATH",
		"-o, --output PATH",
		"-c, --customers PATH",
		"-u, --issuer PATH",
		"-t, --template PATH",
		"invoice.tex in the current directory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestValidateAcceptsShortCustomerAndIssuerFlags(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	exitCode, stdout, stderr := captureRun(t, []string{
		"validate",
		"-i", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Validation OK:") {
		t.Fatalf("stdout %q does not contain validation success output", stdout)
	}
}

func TestRenderDefaultsOutputToInvoiceTex(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	workDir := t.TempDir()
	t.Chdir(workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"render",
		"-i", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Rendered invoice.tex") {
		t.Fatalf("stdout %q does not contain default output path", stdout)
	}
	if _, err := os.Stat(filepath.Join(workDir, "invoice.tex")); err != nil {
		t.Fatalf("default invoice.tex was not created: %v", err)
	}
}

func TestIncrementUpdatesInvoiceNumber(t *testing.T) {
	customersPath, _, _ := writeDraftFixtures(t)
	workDir := t.TempDir()
	invoicePath := filepath.Join(workDir, "invoice.yaml")
	archiveDir := t.TempDir()
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-009
  issue_date: 2026-03-06
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeArchivedInvoiceMarkdown(t, archiveDir, "2026-03-05.md", "CUST-001-011")

	exitCode, stdout, stderr := captureRun(t, []string{
		"increment",
		"-i", invoicePath,
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "CUST-001-009 -> CUST-001-012") {
		t.Fatalf("stdout %q does not contain increment summary", stdout)
	}

	updated, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(updated), "number: CUST-001-012") {
		t.Fatalf("invoice file does not contain incremented number: %q", string(updated))
	}
}

func TestValidateSuggestsGlobalDefaultsWhenSupportFilesMissing(t *testing.T) {
	workDir := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "config-home")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Chdir(workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"validate",
		"-i", filepath.Join(workDir, "invoice.yaml"),
	})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	expected := filepath.Join(configHome, "invox", "customers.yaml")
	if !strings.Contains(stderr, expected) {
		t.Fatalf("stderr %q does not mention global customers path %q", stderr, expected)
	}
}

func TestBuildRejectsNonPDFOutput(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		"-i", invoicePath,
		"-o", filepath.Join(t.TempDir(), "invoice.tex"),
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "-o, --output must end with .pdf") {
		t.Fatalf("stderr %q does not contain output extension error", stderr)
	}
}

func TestBuildLeavesOnlyPDFInWorkingDirectory(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	workDir := t.TempDir()
	fakeBinDir := t.TempDir()
	tectonicPath := filepath.Join(fakeBinDir, "tectonic")
	script := "#!/bin/sh\nset -eu\ninput=\"$1\"\npdf=\"${input%.tex}.pdf\"\n: > \"$pdf\"\n"
	if err := os.WriteFile(tectonicPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(tectonic) returned error: %v", err)
	}

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		"-i", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Built invoice.pdf") {
		t.Fatalf("stdout %q does not contain build output", stdout)
	}

	if _, err := os.Stat(filepath.Join(workDir, "invoice.pdf")); err != nil {
		t.Fatalf("invoice.pdf was not created: %v", err)
	}
	for _, path := range []string{
		filepath.Join(workDir, "invoice.tex"),
		filepath.Join(workDir, "logo.png"),
		filepath.Join(workDir, "fonts"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("unexpected build artifact left behind: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s) returned unexpected error: %v", path, err)
		}
	}
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

func writeContextFixtures(t *testing.T) (string, string, string, string) {
	t.Helper()

	dir := t.TempDir()
	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	invoicePath := filepath.Join(dir, "invoice.yaml")
	templatePath := filepath.Join(dir, "invoice_template.tex")
	fontPath := filepath.Join(dir, "fonts", "Ubuntu-Regular.ttf")

	if err := os.MkdirAll(filepath.Dir(fontPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(fonts) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logo.png"), []byte("logo"), 0o644); err != nil {
		t.Fatalf("WriteFile(logo.png) returned error: %v", err)
	}
	if err := os.WriteFile(fontPath, []byte("font"), 0o644); err != nil {
		t.Fatalf("WriteFile(font) returned error: %v", err)
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
  period_label: Leistungszeitraum
  vat_rate_percent: 20
  paid_amount: 0
line_items:
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

	return customersPath, issuerPath, invoicePath, templatePath
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
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func captureRun(t *testing.T, args []string) (int, string, string) {
	t.Helper()

	if os.Getenv("XDG_CONFIG_HOME") == "" {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config-home"))
	}

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout) returned error: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr) returned error: %v", err)
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	exitCode := Run(args)

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("ReadAll(stdout) returned error: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("ReadAll(stderr) returned error: %v", err)
	}
	_ = stdoutReader.Close()
	_ = stderrReader.Close()

	return exitCode, string(stdoutBytes), string(stderrBytes)
}
