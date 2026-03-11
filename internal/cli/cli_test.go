package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"invox/internal/invoice"
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

func TestConfigOpensMalformedConfigFileForEditing(t *testing.T) {
	configPath := writeConfigFile(t, " numbering:\n  pattern: '{customer_id}-{counter:03}'\n")

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
	if openedPath != configPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, configPath)
	}
	if !strings.Contains(stdout, "Opened "+configPath) {
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Initialized "+configDir) {
		t.Fatalf("stdout %q does not contain initialized config dir", stdout)
	}
	for _, want := range []string{
		"created config.yaml",
		"created customers.yaml",
		"created issuer.yaml",
		"created invoice_defaults.yaml",
		"created template.tex",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
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
	if stderr != "" {
		t.Fatalf("new stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created CUST-001-001.yaml for CUST-001 (CUST-001-001)") {
		t.Fatalf("stdout %q does not contain created invoice summary", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "exists customers.yaml") {
		t.Fatalf("stdout %q does not contain existing customers.yaml status", stdout)
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
		"@@ISSUER_CITY_AND_POSTAL_CODE@@",
		"@@ISSUER_COUNTRY@@",
		"@@CUSTOMER_NAME@@",
		"@@CUSTOMER_STREET@@",
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@",
		"@@CUSTOMER_COUNTRY@@",
		"@@CUSTOMER_VAT_TAX_ID@@",
		"@@CUSTOMER_EMAIL@@",
		"@@INVOICE_NUMBER@@",
		"@@ISSUE_DATE@@",
		"@@DUE_DATE@@",
		"@@PERIOD_LABEL@@",
		"@@LINE_ITEMS_ROWS@@",
		"@@LINE_ITEMS_ROWS_WITH_VAT@@",
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
	for _, want := range []string{
		"multi_vat.tex\t" + filepath.Join(configDir, "multi_vat.tex"),
		"template.tex\t" + filepath.Join(configDir, "template.tex"),
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
	if strings.Contains(stdout, "project.tex\t"+filepath.Join(workDir, "project.tex")) {
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Rendered "+outputPath+" for CUST-001 (CUST-001-001)") {
		t.Fatalf("stdout %q does not contain rendered output path", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created CUST-001-002.yaml for CUST-001") {
		t.Fatalf("stdout %q does not contain success output", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	wantPath := filepath.Join(workDir, "CUST-001-002.yaml")
	if openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, wantPath)
	}
	if !strings.Contains(stdout, "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)") {
		t.Fatalf("stdout %q does not contain success output", stdout)
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

	oldOpenTextFile := openTextFile
	openTextFile = func(path string) error {
		return errors.New("editor unavailable")
	}
	t.Cleanup(func() {
		openTextFile = oldOpenTextFile
	})

	exitCode, stdout, stderr := captureRun(t, []string{
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created CUST-001-007.yaml for CUST-001 (CUST-001-007)") {
		t.Fatalf("stdout %q does not contain customer-specific invoice number", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created "+outputPath+" for CUST-001") {
		t.Fatalf("stdout %q does not contain created output path", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Created CUST-001-003.yaml for CUST-001 (CUST-001-003)") {
		t.Fatalf("stdout %q does not contain cloned invoice output", stdout)
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
		"the input path with .eml extension",
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

	openedPath := ""
	openedSource := ""
	cleanupPath := ""
	oldOpenDocument := openDocument
	oldCleanupOpenedDocument := cleanupOpenedDocument
	oldPreferNativeMailCompose := preferNativeMailCompose
	openDocument = func(path string) error {
		openedPath = path
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		openedSource = string(source)
		return nil
	}
	cleanupOpenedDocument = func(path string) error {
		cleanupPath = path
		return os.Remove(path)
	}
	preferNativeMailCompose = false
	t.Cleanup(func() {
		openDocument = oldOpenDocument
		cleanupOpenedDocument = oldCleanupOpenedDocument
		preferNativeMailCompose = oldPreferNativeMailCompose
	})

	exitCode, stdout, stderr := captureRun(t, []string{
		"email",
		customInvoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if openedPath != outputPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, outputPath)
	}
	if cleanupPath != outputPath {
		t.Fatalf("cleanupPath = %q, want %q", cleanupPath, outputPath)
	}
	if !strings.Contains(stdout, "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example") {
		t.Fatalf("stdout %q does not contain email summary", stdout)
	}
	if !strings.Contains(openedSource, `filename="BL00210001.pdf"`) {
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

	openedPath := ""
	openedSource := ""
	cleanupPath := ""
	oldOpenDocument := openDocument
	oldCleanupOpenedDocument := cleanupOpenedDocument
	oldPreferNativeMailCompose := preferNativeMailCompose
	openDocument = func(path string) error {
		openedPath = path
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		openedSource = string(source)
		return nil
	}
	cleanupOpenedDocument = func(path string) error {
		cleanupPath = path
		return os.Remove(path)
	}
	preferNativeMailCompose = false
	t.Cleanup(func() {
		openDocument = oldOpenDocument
		cleanupOpenedDocument = oldCleanupOpenedDocument
		preferNativeMailCompose = oldPreferNativeMailCompose
	})

	exitCode, stdout, stderr := captureRun(t, []string{
		"email",
		pdfPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if openedPath != outputPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, outputPath)
	}
	if cleanupPath != outputPath {
		t.Fatalf("cleanupPath = %q, want %q", cleanupPath, outputPath)
	}
	if !strings.Contains(stdout, "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example") {
		t.Fatalf("stdout %q does not contain email summary", stdout)
	}
	if !strings.Contains(openedSource, `filename="BL00210001.pdf"`) {
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

	openedPath := ""
	cleanupPath := ""
	oldOpenDocument := openDocument
	oldCleanupOpenedDocument := cleanupOpenedDocument
	oldPreferNativeMailCompose := preferNativeMailCompose
	openDocument = func(path string) error {
		openedPath = path
		return nil
	}
	cleanupOpenedDocument = func(path string) error {
		cleanupPath = path
		return os.Remove(path)
	}
	preferNativeMailCompose = false
	t.Cleanup(func() {
		openDocument = oldOpenDocument
		cleanupOpenedDocument = oldCleanupOpenedDocument
		preferNativeMailCompose = oldPreferNativeMailCompose
	})

	exitCode, stdout, stderr := captureRun(t, []string{
		"email",
		pdfPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if openedPath != outputPath {
		t.Fatalf("openedPath = %q, want %q", openedPath, outputPath)
	}
	if cleanupPath != outputPath {
		t.Fatalf("cleanupPath = %q, want %q", cleanupPath, outputPath)
	}
	if !strings.Contains(stdout, "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example") {
		t.Fatalf("stdout %q does not contain email summary", stdout)
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

	oldPreferNativeMailCompose := preferNativeMailCompose
	oldOpenNativeEmailDraft := openNativeEmailDraft
	oldOpenDocument := openDocument
	oldCleanupOpenedDocument := cleanupOpenedDocument
	preferNativeMailCompose = true

	var opened invoice.EmailMessage
	openNativeEmailDraft = func(message invoice.EmailMessage) error {
		opened = message
		return nil
	}
	openDocument = func(path string) error {
		t.Fatalf("openDocument(%q) should not be called when native compose is enabled", path)
		return nil
	}
	cleanupOpenedDocument = func(path string) error {
		t.Fatalf("cleanupOpenedDocument(%q) should not be called when native compose is enabled", path)
		return nil
	}
	t.Cleanup(func() {
		preferNativeMailCompose = oldPreferNativeMailCompose
		openNativeEmailDraft = oldOpenNativeEmailDraft
		openDocument = oldOpenDocument
		cleanupOpenedDocument = oldCleanupOpenedDocument
	})

	exitCode, stdout, stderr := captureRun(t, []string{
		"email",
		customInvoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if opened.Recipient != "office@appsters.example" {
		t.Fatalf("Recipient = %q, want %q", opened.Recipient, "office@appsters.example")
	}
	if opened.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", opened.Subject, "Invoice CUST-001-001")
	}
	if opened.AttachmentPath != pdfPath {
		t.Fatalf("AttachmentPath = %q, want %q", opened.AttachmentPath, pdfPath)
	}
	if !strings.Contains(opened.Body, "Please find attached invoice CUST-001-001.") {
		t.Fatalf("Body %q does not contain the default invoice text", opened.Body)
	}
	if strings.Contains(opened.Body, "\r") {
		t.Fatalf("Body = %q, want LF-only newlines", opened.Body)
	}
	if !strings.Contains(stdout, "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example") {
		t.Fatalf("stdout %q does not contain email summary", stdout)
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
	fakeBinDir := t.TempDir()
	tectonicPath := filepath.Join(fakeBinDir, "tectonic")
	script := "#!/bin/sh\nset -eu\ninput=\"$1\"\npdf=\"${input%.tex}.pdf\"\n: > \"$pdf\"\n"
	if err := os.WriteFile(tectonicPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(tectonic) returned error: %v", err)
	}

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Built "+customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))]+".pdf") {
		t.Fatalf("stdout %q does not contain build output", stdout)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
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
	fakeBinDir := t.TempDir()
	tectonicPath := filepath.Join(fakeBinDir, "tectonic")
	script := "#!/bin/sh\nset -eu\ninput=\"$1\"\npdf=\"${input%.tex}.pdf\"\n: > \"$pdf\"\n"
	if err := os.WriteFile(tectonicPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(tectonic) returned error: %v", err)
	}

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	outputPath := customInvoicePath[:len(customInvoicePath)-len(filepath.Ext(customInvoicePath))] + ".pdf"
	archivePath := filepath.Join(archiveDir, filepath.Base(customInvoicePath))
	if !strings.Contains(stdout, "Built "+outputPath) {
		t.Fatalf("stdout %q does not contain build output", stdout)
	}
	if !strings.Contains(stdout, "Archived "+customInvoicePath+" -> "+archivePath) {
		t.Fatalf("stdout %q does not contain archive output", stdout)
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
	fakeBinDir := t.TempDir()
	tectonicPath := filepath.Join(fakeBinDir, "tectonic")
	script := "#!/bin/sh\nset -eu\nexit 1\n"
	if err := os.WriteFile(tectonicPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(tectonic) returned error: %v", err)
	}

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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
	fakeBinDir := t.TempDir()
	tectonicPath := filepath.Join(fakeBinDir, "tectonic")
	script := "#!/bin/sh\nset -eu\ninput=\"$1\"\npdf=\"${input%.tex}.pdf\"\n: > \"$pdf\"\n"
	if err := os.WriteFile(tectonicPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(tectonic) returned error: %v", err)
	}

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	archivePath := filepath.Join(archiveDir, filepath.Base(invoicePath))
	if !strings.Contains(stdout, "Archived "+invoicePath+" -> "+archivePath) {
		t.Fatalf("stdout %q does not contain archive summary", stdout)
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	editedPath := filepath.Join(workDir, "2026-03-06.yaml")
	if !strings.Contains(stdout, "Editing "+archivedPath+" -> 2026-03-06.yaml") {
		t.Fatalf("stdout %q does not contain edit summary", stdout)
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
	if stderr != "" {
		t.Fatalf("edit stderr = %q, want empty", stderr)
	}
	if stdout == "" {
		t.Fatal("edit stdout should not be empty")
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
	})
	if exitCode != 0 {
		t.Fatalf("archive exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("archive stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Archived 2026-03-06.yaml -> "+archivedPath) {
		t.Fatalf("stdout %q does not contain re-archive summary", stdout)
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
