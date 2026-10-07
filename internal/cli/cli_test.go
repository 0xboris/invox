package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
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
		"Customer fields:",
		"<customer>.tax.default_vat_rate",
		"<customer>.billing.send_invoice_to",
		"<customer>.legal_company_name",
		"customers.yaml example:",
		"CUST-001:",
		"send_invoice_to: accounting@appsters.example",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestRootHelpShowsDocumentationTopics(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Documentation topics:",
		"invox help config",
		"invox help customers",
		"invox help issuer",
		"invox help defaults",
		"invox help template",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestConfigOpensConfigFile(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	f, stub := testFactory(t)
	openedPath := expectEditor(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	wantPath := filepath.Join(configHome, "invox", "config.yaml")
	if *openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, wantPath)
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
	if want := "Opened " + wantPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestConfigOpensMalformedConfigFileForEditing(t *testing.T) {
	configPath := writeConfigFile(t, " numbering:\n  pattern: '{customer_id}-{counter:03}'\n")

	f, stub := testFactory(t)
	openedPath := expectEditor(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if *openedPath != configPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, configPath)
	}
	if want := "Opened " + configPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
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
		"email.subject",
		"email.body",
		"email template placeholders:",
		"{email_greeting}",
		"{contact_person}",
		"{outstanding_amount}",
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

func TestInitCreatesStarterFilesAndAllowsNewWithGlobalDefaults(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	exitCode, stdout, stderr := captureRun(t, []string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\ncreated customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	for _, file := range []struct {
		name string
		want string
	}{
		{name: "config.yaml", want: "# Invox user configuration."},
		{name: "customers.yaml", want: "CUST-001:"},
		{name: "issuer.yaml", want: "vat_label: VAT"},
		{name: "invoice_defaults.yaml", want: "status: draft"},
		{name: "template.tex", want: "\\documentclass"},
	} {
		path := filepath.Join(configDir, file.name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", path, err)
		}
		if !strings.Contains(string(source), file.want) {
			t.Fatalf("%s does not contain %q:\n%s", path, file.want, string(source))
		}
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, stdout, stderr = captureRun(t, []string{"new", "CUST-001"})
	if exitCode != 0 {
		t.Fatalf("new exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-001.yaml for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-001.yaml\n" {
		t.Fatalf("new stdout = %q, want %q", stdout, "CUST-001-001.yaml\n")
	}
	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-001.yaml")); err != nil {
		t.Fatalf("starter config should allow invoice creation: %v", err)
	}
}

func TestInitDoesNotOverwriteExistingSupportFiles(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}

	customCustomers := "CUSTOM:\n  name: Custom Customer\n"
	customersPath := filepath.Join(configDir, "customers.yaml")
	if err := os.WriteFile(customersPath, []byte(customCustomers), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\nexists customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	source, err := os.ReadFile(customersPath)
	if err != nil {
		t.Fatalf("ReadFile(customers.yaml) returned error: %v", err)
	}
	if string(source) != customCustomers {
		t.Fatalf("customers.yaml was overwritten:\n%s", string(source))
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
		"email.subject",
		"email.body",
		"email template placeholders:",
		"{email_greeting}",
		"{contact_person}",
		"Template:",
		"# paths:",
		"# archive:",
		"# email:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestHelpCustomersShowsCustomersDocumentation(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"help", "customers"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"customers.yaml reference.",
		"invox help customers",
		"invox customer config",
		"Formatting:",
		"Top-level customer IDs must start at column 1 with no leading spaces.",
		"Customer fields:",
		"Preferred fields:",
		"<customer>.name",
		"<customer>.billing.send_invoice_to",
		"Alternate supported paths:",
		"<customer>.legal_company_name",
		"Rules:",
		"Email lookup order is billing.send_invoice_to, billing.email, then email.",
		"billing.currency defaults to EUR.",
		"customers.yaml example:",
		"CUST-001:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestHelpIssuerShowsIssuerDocumentation(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"help", "issuer"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"issuer.yaml reference.",
		"invox help issuer",
		"Formatting:",
		"Top-level keys must start at column 1 with no leading spaces.",
		"Issuer fields:",
		"Required company fields:",
		"company.legal_company_name",
		"company.email",
		"Required payment fields:",
		"payment.due_days",
		"payment.payment_terms_text",
		"Optional payment fields:",
		"payment.vat_label",
		"payment.epc_qr.label",
		"payment.epc_qr.text",
		"Rules:",
		"payment.due_days must be a non-negative integer.",
		"EPC QR generation requires a valid SEPA-scope payment.iban.",
		"issuer.yaml example:",
		"company:",
		"payment:",
		"# name: Boris Consulting",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestHelpDefaultsShowsInvoiceDefaultsDocumentation(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"help", "defaults"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invoice_defaults.yaml reference.",
		"invox help defaults",
		"invox help invoice-defaults",
		"Formatting:",
		"Top-level keys must start at column 1 with no leading spaces.",
		"invoice_defaults.yaml fields:",
		"Top-level keys:",
		"invoice.number",
		"invoice.vat_percent",
		"positions[].unit_price",
		"Unsupported legacy keys:",
		"line_items",
		"Rules:",
		"`new` sets customer_id, invoice.number, invoice.issue_date, invoice.due_date, invoice.status, and invoice.paid_amount.",
		"invoice_defaults.yaml example:",
		"positions:",
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

	f, stub := testFactory(t)
	openedPath := expectEditor(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"customer",
		"config",
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if *openedPath != customersPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, customersPath)
	}
	if want := "Opened " + customersPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
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
		"<customer>.billing.send_invoice_to",
		"<customer>.tax.default_vat_rate",
		"<customer>.billing.currency",
		"<customer>.numbering.code",
		"<customer>.numbering.start",
		"invox customer config",
		"customers.yaml example:",
		"# legal_company_name: Appsters GmbH",
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

func TestTemplateHelpShowsTemplateSubcommands(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"template", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Description:",
		"Author and discover the LaTeX templates used by render and build.",
		"invox template <subcommand> [options]",
		"list          List available invoice templates",
		"Important rules:",
		"Placeholder names are case-sensitive and must match exactly.",
		"Structured placeholders:",
		"Template workflow:",
		"Available .tex placeholders:",
		"@@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@ repeats a custom snippet once per position.",
		"Custom line-item block example:",
		"@@LINE_ITEM_NAME@@ & @@LINE_ITEM_UNIT_PRICE@@ & @@LINE_ITEM_VAT_RATE@@ & @@LINE_ITEM_LINE_TOTAL@@\\\\",
		"Use @@VAT_LABEL@@ anywhere you want the same VAT label text in the template.",
		"render -i invoice.yaml -t multi_vat.tex",
		"invox template list --names",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
	for _, placeholder := range []string{
		"@@ISSUER_NAME@@",
		"@@ISSUER_COMPANY_REG_NO@@",
		"@@ISSUER_VAT_TAX_ID@@",
		"@@ISSUER_WEBSITE@@",
		"@@ISSUER_EMAIL@@",
		"@@ISSUER_STREET@@",
		"@@ISSUER_CITY@@",
		"@@ISSUER_POSTAL_CODE@@",
		"@@ISSUER_COUNTRY@@",
		"@@CUSTOMER_NAME@@",
		"@@CUSTOMER_STREET@@",
		"@@CUSTOMER_CITY@@",
		"@@CUSTOMER_POSTAL_CODE@@",
		"@@CUSTOMER_COUNTRY@@",
		"@@CUSTOMER_VAT_TAX_ID@@",
		"@@CUSTOMER_EMAIL@@",
		"@@INVOICE_NUMBER@@",
		"@@ISSUE_DATE@@",
		"@@DUE_DATE@@",
		"@@PERIOD_LABEL@@",
		"@@LINE_ITEMS_ROWS@@",
		"@@LINE_ITEMS_ROWS_WITH_VAT@@",
		"@@LINE_ITEMS_BEGIN@@",
		"@@LINE_ITEMS_END@@",
		"@@LINE_ITEM_NAME@@",
		"@@LINE_ITEM_DESCRIPTION@@",
		"@@LINE_ITEM_UNIT_PRICE@@",
		"@@LINE_ITEM_QUANTITY@@",
		"@@LINE_ITEM_VAT_RATE@@",
		"@@LINE_ITEM_LINE_TOTAL@@",
		"@@LINE_ITEM_RULE@@",
		"@@SUBTOTAL@@",
		"@@VAT_SUMMARY_ROWS@@",
		"@@TOTAL@@",
		"@@PAID_AMOUNT@@",
		"@@OUTSTANDING_AMOUNT@@",
		"@@INVOICE_TOTAL@@",
		"@@OUTSTANDING_TOTAL@@",
		"@@PAYMENT_TERMS_TEXT@@",
		"@@VAT_LABEL@@",
		"@@BANK_NAME@@",
		"@@IBAN@@",
		"@@BIC@@",
		"@@EPC_QR_AVAILABLE@@",
		"@@EPC_QR_LABEL@@",
		"@@EPC_QR_CODE@@",
	} {
		if !strings.Contains(stdout, placeholder) {
			t.Fatalf("stdout %q does not contain placeholder %q", stdout, placeholder)
		}
	}
}

func TestTemplateListPrintsAvailableTemplates(t *testing.T) {
	configPath := writeConfigFile(t, "")
	configDir := filepath.Dir(configPath)
	workDir := t.TempDir()
	chdirForTest(t, workDir)

	for _, path := range []string{
		filepath.Join(configDir, "multi_vat.tex"),
		filepath.Join(configDir, "template.tex"),
		filepath.Join(workDir, "project.tex"),
	} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	exitCode, stdout, stderr := captureRun(t, []string{"template", "list"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	// Piped output escapes the backslashes in Windows paths.
	tsvPath := func(path string) string { return strings.ReplaceAll(path, `\`, `\\`) }
	for _, want := range []string{
		"multi_vat.tex\t" + tsvPath(filepath.Join(configDir, "multi_vat.tex")),
		"template.tex\t" + tsvPath(filepath.Join(configDir, "template.tex")),
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
	if strings.Contains(stdout, "project.tex\t"+tsvPath(filepath.Join(workDir, "project.tex"))) {
		t.Fatalf("stdout %q should not contain project template outside default template dir", stdout)
	}
}

func TestRenderAcceptsTemplateFilenameFromGlobalConfig(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	configPath := writeConfigFile(t, "")
	configDir := filepath.Dir(configPath)

	templatePath := filepath.Join(configDir, "multi_vat.tex")
	if err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
Invoice @@INVOICE_NUMBER@@
Customer @@CUSTOMER_NAME@@
@@VAT_SUMMARY_ROWS@@
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "template.tex"), []byte("starter\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(template.tex) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	exitCode, stdout, stderr := captureRun(t, []string{
		"render",
		"-i", invoicePath,
		"-o", outputPath,
		"-c", customersPath,
		"-u", issuerPath,
		"-t", "multi_vat.tex",
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Rendered " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	for _, want := range []string{
		"Invoice CUST-001-001",
		"Customer Appsters GmbH",
		"VAT (20\\%):",
	} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("rendered output %q does not contain %q", string(rendered), want)
		}
	}
}

func TestCompletionZshOutputsTemplateAutocomplete(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"completion", "zsh"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"#compdef invox",
		"template list --names",
		"_invox_template_values",
		"_invox_invoice_files()",
		"_invox_shift_words()",
		"_invox_shift_words 1",
		"_invox_shift_words 2",
		"if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then",
		"{-t+,--template=}",
		"(-i --input)1::invoice:_files",
		"compdef _invox invox",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
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
		"-e, --edit",
		"--from-last",
		"<invoice.number>.yaml in the current directory",
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
	chdirForTest(t, workDir)

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
	if want := "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-002.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-002.yaml\n")
	}
	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-002.yaml")); err != nil {
		t.Fatalf("default invoice file was not created: %v", err)
	}
}

func TestNewEditOpensCreatedInvoiceFile(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	workDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	chdirForTest(t, workDir)

	f, stub := testFactory(t)
	openedPath := expectEditor(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"new",
		"CUST-001",
		"-e",
		"-c", customersPath,
		"-u", issuerPath,
		"-s", defaultsPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	wantPath := filepath.Join(workDir, "CUST-001-002.yaml")
	if *openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, wantPath)
	}
	if want := "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-002.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-002.yaml\n")
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("edited invoice file was not created: %v", err)
	}
}

func TestNewEditReportsFailureAfterCreatingInvoiceFile(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	workDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	chdirForTest(t, workDir)

	f, stub := testFactory(t)
	expectEditor(stub, errors.New("editor unavailable"))

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"new",
		"CUST-001",
		"--edit",
		"-c", customersPath,
		"-u", issuerPath,
		"-s", defaultsPath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "created CUST-001-002.yaml but failed to open it: editor unavailable") {
		t.Fatalf("stderr %q does not contain editor error", stderr)
	}

	wantPath := filepath.Join(workDir, "CUST-001-002.yaml")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("invoice file should still have been created: %v", err)
	}
}

func TestNewUsesCustomerSpecificStartFromCustomersFile(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixturesWithCustomerStart(t, "7")
	workDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	chdirForTest(t, workDir)

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
	if want := "Created CUST-001-007.yaml for CUST-001 (CUST-001-007)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-007.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-007.yaml\n")
	}
}

func TestNewAcceptsInlineLongFlagsAfterCustomerID(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	outputPath := filepath.Join(t.TempDir(), "out.yaml")

	exitCode, stdout, stderr := captureRun(t, []string{
		"new",
		"CUST-001",
		"--output=" + outputPath,
		"--customers=" + customersPath,
		"--issuer=" + issuerPath,
		"--source=" + defaultsPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file was not created: %v", err)
	}
}

func TestNewFromLastUsesLatestArchivedInvoiceForCustomer(t *testing.T) {
	customersPath, issuerPath, _ := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	if err := os.WriteFile(filepath.Join(archiveDir, "2026-03-01.yaml"), []byte(strings.TrimSpace(`
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
    description: Keep me old
    unit_price: 50
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(older archived invoice) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "2026-03-08.yaml"), []byte(strings.TrimSpace(`
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
    description: Keep me latest
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(latest archived invoice) returned error: %v", err)
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"new",
		"CUST-001",
		"--from-last",
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-003.yaml for CUST-001 (CUST-001-003)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-003.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-003.yaml\n")
	}

	createdPath := filepath.Join(workDir, "CUST-001-003.yaml")
	createdSource, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatalf("ReadFile(createdPath) returned error: %v", err)
	}
	createdText := string(createdSource)
	for _, want := range []string{
		"number: CUST-001-003",
		"status: draft",
		"paid_amount: \"0\"",
		"period: March 2026",
		"name: Latest position",
		"description: Keep me latest",
		"unit_price: 120",
		"quantity: 2",
	} {
		if !strings.Contains(createdText, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, createdText)
		}
	}
	for _, forbidden := range []string{
		"Older position",
		"Keep me old",
		"paid_amount: 999",
		"status: archived",
	} {
		if strings.Contains(createdText, forbidden) {
			t.Fatalf("created invoice should not contain %q:\n%s", forbidden, createdText)
		}
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
	if want := "error: missing required flags: -i, --input\nRun 'invox render --help' for usage.\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
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
		"schema/docs: run `invox help customers`",
		"schema/docs: run `invox help issuer`",
		"invoice.tex in the current directory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestNewHelpShowsSupportFileDocumentationHints(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"new", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"-c, --customers PATH",
		"-u, --issuer PATH",
		"-s, --source PATH",
		"schema/docs: run `invox help customers`",
		"schema/docs: run `invox help issuer`",
		"schema/docs: run `invox help defaults`",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestEmailHelpShowsDraftOutputAndFlags(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"email", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml, INVOICE.pdf, or -i, --input PATH",
		"-p, --pdf PATH",
		"-o, --output PATH",
		"--to EMAIL",
		"--subject TEXT",
		"--force",
		"<input name>.eml in a new temporary directory, removed after 24 hours",
		"Accepts either the invoice YAML file or the built PDF as input.",
		"The PDF lookup checks next to the PDF first, then archive.dir.",
		"Requires invoice.status to be built or archived and the PDF attachment to exist.",
		"On macOS, opens an editable compose window in Apple Mail with the PDF attached.",
		"If -o is set, or on non-macOS platforms, writes a .eml draft file and opens it.",
		"Does not send the email and does not change invoice.status.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestEmailDefaultsDraftPathFromInputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	f, stub := testFactory(t)
	openedPath := expectOpener(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"email",
		customInvoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	openedSource, err := os.ReadFile(*openedPath)
	if err != nil {
		t.Fatalf("ReadFile(openedPath) returned error: %v", err)
	}
	if !strings.Contains(string(openedSource), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft email does not contain attached PDF filename:\n%s", openedSource)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestEmailAcceptsPDFInputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	f, stub := testFactory(t)
	openedPath := expectOpener(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"email",
		pdfPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	openedSource, err := os.ReadFile(*openedPath)
	if err != nil {
		t.Fatalf("ReadFile(openedPath) returned error: %v", err)
	}
	if !strings.Contains(string(openedSource), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft email does not contain attached PDF filename:\n%s", openedSource)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestEmailFindsInvoiceYAMLInArchiveDirForPDFInput(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedInvoicePath := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedInvoicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(archivedInvoicePath)) returned error: %v", err)
	}
	if err := os.WriteFile(archivedInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedInvoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	f, stub := testFactory(t)
	openedPath := expectOpener(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"email",
		pdfPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestSendAliasUsesEmailCommand(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"send", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox email (INVOICE.yaml | INVOICE.pdf | -i INPUT)",
		"On macOS, opens an editable compose window in Apple Mail with the PDF attached.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestEmailUsesEditableNativeComposeByDefault(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	f, stub := testFactory(t)
	f.Mailer = applemail.New(stub, f.IOStreams)
	var opened applemail.Message
	stub.Register("osascript", func(cmd run.Cmd) error {
		message := cmd.Args[slices.Index(cmd.Args, "--")+1:]
		opened = applemail.Message{To: message[0], Subject: message[1], Body: message[2], Attachment: message[3], Sender: message[4]}
		return nil
	})

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{
		"email",
		customInvoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if opened.To != "office@appsters.example" {
		t.Fatalf("To = %q, want %q", opened.To, "office@appsters.example")
	}
	if opened.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", opened.Subject, "Invoice CUST-001-001")
	}
	if opened.Attachment != pdfPath {
		t.Fatalf("Attachment = %q, want %q", opened.Attachment, pdfPath)
	}
	if !strings.Contains(opened.Body, "Please find attached invoice CUST-001-001.") {
		t.Fatalf("Body %q does not contain the default invoice text", opened.Body)
	}
	if strings.Contains(opened.Body, "\r") {
		t.Fatalf("Body = %q, want LF-only newlines", opened.Body)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(filepath.Join(inputDir, "BL00210001.eml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(default .eml path) error = %v, want not exists", err)
	}
}

func TestBuildHelpShowsInputBasedDefaultOutput(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"build", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml or -i, --input PATH",
		"-o, --output PATH",
		"--archive",
		"-c, --customers PATH",
		"-u, --issuer PATH",
		"-t, --template PATH",
		"the input path with .pdf extension",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestBuildRequiresPositionalOrFlagInput(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"build"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required input: INVOICE.yaml or -i, --input") {
		t.Fatalf("stderr %q does not contain missing input message", stderr)
	}
}

func TestArchiveHelpShowsShortFlags(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"archive", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml or -i, --input PATH",
		"invox archive (INVOICE.yaml | -i INVOICE.yaml)",
		"invox archive invoice.yaml",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveListHelpShowsOutputFormat(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"archive", "list", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox archive list",
		"FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveEditHelpShowsUsage(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"archive", "edit", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox archive edit FILENAME",
		"archive.dir",
		"invoice.status set to editing",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveRequiresPositionalOrFlagInput(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"archive"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required input: INVOICE.yaml or -i, --input") {
		t.Fatalf("stderr %q does not contain missing input message", stderr)
	}
}

func TestArchiveListPrintsArchivedInvoices(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	yamlArchivePath := filepath.Join(archiveDir, "2026-03-06.yaml")
	if err := os.WriteFile(yamlArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-YAML
invoice:
  number: CUST-YAML-001
  issue_date: 2026-03-06
  status: archived
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(yamlArchivePath) returned error: %v", err)
	}

	markdownArchivePath := filepath.Join(archiveDir, "2026-03-05.md")
	if err := os.WriteFile(markdownArchivePath, []byte(strings.Join([]string{
		"---",
		"customer_id: CUST-MD",
		"invoice:",
		"  number: CUST-MD-001",
		"  issue_date: 2026-03-05",
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(markdownArchivePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "list"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := strings.Join([]string{
		"2026-03-05.md\tCUST-MD\t2026-03-05\tarchived",
		"2026-03-06.yaml\tCUST-YAML\t2026-03-06\tarchived",
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
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
	if want := "Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestRenderDefaultsOutputToInvoiceTex(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	workDir := t.TempDir()
	chdirForTest(t, workDir)

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
	if want := "Rendered invoice.tex for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "invoice.tex\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "invoice.tex\n")
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
	if want := "Incremented " + invoicePath + " for CUST-001: CUST-001-009 -> CUST-001-012\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != invoicePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, invoicePath+"\n")
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
	chdirForTest(t, workDir)

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

func TestBuildDefaultsPDFPathFromInputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	workDir := t.TempDir()
	installFakeTectonic(t, fakeTectonicWritePDF)

	chdirForTest(t, workDir)

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		customInvoicePath,
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	if want := "Built " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("default output PDF was not created: %v", err)
	}
	updatedInvoice, err := os.ReadFile(customInvoicePath)
	if err != nil {
		t.Fatalf("ReadFile(customInvoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(updatedInvoice), "status: built") {
		t.Fatalf("invoice file was not marked built:\n%s", string(updatedInvoice))
	}
	if _, err := os.Stat(filepath.Join(workDir, "invoice.pdf")); err == nil {
		t.Fatal("build should not write invoice.pdf into the current working directory")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(workDir/invoice.pdf) returned unexpected error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(inputDir, "BL00210001.tex"),
		filepath.Join(inputDir, "logo.png"),
		filepath.Join(inputDir, "fonts"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("unexpected build artifact left behind: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s) returned unexpected error: %v", path, err)
		}
	}
}

func TestBuildWithArchiveMovesInvoiceToArchiveDir(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210002.yaml")
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		customInvoicePath,
		"--archive",
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	archivePath := filepath.Join(archiveDir, filepath.Base(customInvoicePath))
	if want := "Built " + outputPath + " for CUST-001 (CUST-001-001)\nArchived " + customInvoicePath + " -> " + archivePath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output PDF was not created: %v", err)
	}
	if _, err := os.Stat(customInvoicePath); err == nil {
		t.Fatalf("source invoice should have been archived: %s", customInvoicePath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(customInvoicePath) returned unexpected error: %v", err)
	}
	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if !strings.Contains(string(archivedSource), "status: archived") {
		t.Fatalf("archived invoice does not contain archived status:\n%s", string(archivedSource))
	}
}

func TestBuildDoesNotMarkInvoiceBuiltWhenPDFBuildFails(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicFail)

	sourceBefore, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		"-i", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr == "" {
		t.Fatal("stderr should contain build failure output")
	}

	sourceAfter, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if string(sourceAfter) != string(sourceBefore) {
		t.Fatalf("failed build should not modify invoice file:\nbefore:\n%s\nafter:\n%s", string(sourceBefore), string(sourceAfter))
	}
	if strings.Contains(string(sourceAfter), "status: built") {
		t.Fatalf("failed build should not mark invoice built:\n%s", string(sourceAfter))
	}
}

func TestBuildWithArchiveLeavesInvoiceBuiltWhenArchiveFails(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210003.yaml")
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(customInvoicePath, source, 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	archivePath := filepath.Join(archiveDir, filepath.Base(customInvoicePath))
	if err := os.WriteFile(archivePath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"build",
		customInvoicePath,
		"--archive",
		"-c", customersPath,
		"-u", issuerPath,
		"-t", templatePath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "built "+customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))]+".pdf but failed to archive "+customInvoicePath) {
		t.Fatalf("stderr %q does not contain archive failure context", stderr)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output PDF should still exist: %v", err)
	}
	sourceAfter, err := os.ReadFile(customInvoicePath)
	if err != nil {
		t.Fatalf("ReadFile(customInvoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(sourceAfter), "status: built") {
		t.Fatalf("invoice should remain built after archive failure:\n%s", string(sourceAfter))
	}
	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if string(archivedSource) != "existing" {
		t.Fatalf("existing archive file should remain unchanged, got %q", string(archivedSource))
	}
}

func TestArchiveMovesBuiltInvoiceToArchiveDir(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: built
  period: Leistungszeitraum
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"archive",
		invoicePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	archivePath := filepath.Join(archiveDir, filepath.Base(invoicePath))
	if want := "Archived " + invoicePath + " -> " + archivePath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != archivePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, archivePath+"\n")
	}
	if _, err := os.Stat(invoicePath); err == nil {
		t.Fatalf("source invoice should have been removed: %s", invoicePath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(invoicePath) returned unexpected error: %v", err)
	}

	archivedSource, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archivePath) returned error: %v", err)
	}
	if !strings.Contains(string(archivedSource), "status: archived") {
		t.Fatalf("archived invoice does not contain archived status:\n%s", string(archivedSource))
	}
}

func TestArchiveEditCopiesArchivedInvoiceToCurrentDir(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedPath := filepath.Join(archiveDir, "customer-a", "2026-03-06.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive subdir) returned error: %v", err)
	}
	if err := os.WriteFile(archivedPath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: archived
  period: March 2026
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedPath) returned error: %v", err)
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"archive",
		"edit",
		"customer-a/2026-03-06.yaml",
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	editedPath := filepath.Join(workDir, "2026-03-06.yaml")
	if want := "Editing " + archivedPath + " -> 2026-03-06.yaml\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "2026-03-06.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "2026-03-06.yaml\n")
	}
	editedSource, err := os.ReadFile(editedPath)
	if err != nil {
		t.Fatalf("ReadFile(editedPath) returned error: %v", err)
	}
	editedText := string(editedSource)
	for _, want := range []string{
		"status: editing",
		"_invox:",
		"archive_path: customer-a/2026-03-06.yaml",
	} {
		if !strings.Contains(editedText, want) {
			t.Fatalf("edited invoice does not contain %q:\n%s", want, editedText)
		}
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived invoice should remain in place: %v", err)
	}
}

func TestArchiveReplacesEditedArchivedInvoice(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedPath := filepath.Join(archiveDir, "2026-03-06.yaml")
	if err := os.WriteFile(archivedPath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: archived
  period: March 2026
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Original archive
    unit_price: 100
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedPath) returned error: %v", err)
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)

	exitCode, stdout, stderr := captureRun(t, []string{
		"archive",
		"edit",
		"2026-03-06.yaml",
	})
	if exitCode != 0 {
		t.Fatalf("edit exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Editing " + archivedPath + " -> 2026-03-06.yaml\n"; stderr != want {
		t.Fatalf("edit stderr = %q, want %q", stderr, want)
	}
	if stdout != "2026-03-06.yaml\n" {
		t.Fatalf("edit stdout = %q, want %q", stdout, "2026-03-06.yaml\n")
	}

	editedPath := filepath.Join(workDir, "2026-03-06.yaml")
	editedSource, err := os.ReadFile(editedPath)
	if err != nil {
		t.Fatalf("ReadFile(editedPath) returned error: %v", err)
	}
	mutated := strings.Replace(string(editedSource), "Original archive", "Updated archive", 1)
	if err := os.WriteFile(editedPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(editedPath) returned error: %v", err)
	}

	exitCode, stdout, stderr = captureRun(t, []string{
		"archive",
		editedPath,
		"--yes",
	})
	if exitCode != 0 {
		t.Fatalf("archive exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.HasPrefix(stderr, "Replaced archived invoice "+archivedPath+"; previous version kept at ") {
		t.Fatalf("archive stderr = %q, want replacement notice", stderr)
	}
	if !strings.HasSuffix(stderr, "\nArchived 2026-03-06.yaml -> "+archivedPath+"\n") {
		t.Fatalf("archive stderr %q does not end with the re-archive summary", stderr)
	}
	if stdout != archivedPath+"\n" {
		t.Fatalf("archive stdout = %q, want %q", stdout, archivedPath+"\n")
	}
	if _, err := os.Stat(editedPath); err == nil {
		t.Fatalf("edited working copy should have been removed: %s", editedPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(editedPath) returned unexpected error: %v", err)
	}

	archivedSource, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("ReadFile(archivedPath) returned error: %v", err)
	}
	archivedText := string(archivedSource)
	for _, want := range []string{
		"status: archived",
		"Updated archive",
	} {
		if !strings.Contains(archivedText, want) {
			t.Fatalf("archived invoice does not contain %q:\n%s", want, archivedText)
		}
	}
	for _, forbidden := range []string{
		"status: editing",
		"_invox:",
	} {
		if strings.Contains(archivedText, forbidden) {
			t.Fatalf("archived invoice should not contain %q:\n%s", forbidden, archivedText)
		}
	}
}

func TestArchiveRejectsInvoiceWithoutBuiltStatus(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: draft
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"archive",
		invoicePath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "invoice.status must be `built` before archiving") {
		t.Fatalf("stderr %q does not contain status validation", stderr)
	}
	if _, err := os.Stat(invoicePath); err != nil {
		t.Fatalf("source invoice should remain in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, filepath.Base(invoicePath))); err == nil {
		t.Fatal("invoice should not be moved into archive dir")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(archivePath) returned unexpected error: %v", err)
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
  email_greeting: Dear Jane Doe,
  contact_person: Jane Doe
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

// quoteYAMLString single-quotes value for YAML, where backslashes (as in
// Windows paths) are literal.
func quoteYAMLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func captureRun(t *testing.T, args []string) (int, string, string) {
	t.Helper()

	ios, _, _, _ := iostreams.Test()
	return captureRunStreams(t, ios, args)
}

// captureRunStreams runs args with ios, which must come from iostreams.Test,
// and returns the exit code, stdout and stderr.
func captureRunStreams(t *testing.T, ios *iostreams.IOStreams, args []string) (int, string, string) {
	t.Helper()

	isolateUserDirs(t)
	exitCode := Main(args, cmdutil.NewFactory(ios, run.Exec{}, env.System()))
	return exitCode, ios.Out.(*bytes.Buffer).String(), ios.ErrOut.(*bytes.Buffer).String()
}
