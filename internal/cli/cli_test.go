package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestRootHelpShowsDocumentationTopics(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Help topics:\n",
		"  config       config.yaml keys, precedence, and email placeholders\n",
		"  customers    customers.yaml fields, aliases, and example\n",
		"  issuer       issuer.yaml fields, validation rules, and example\n",
		"  defaults     invoice_defaults.yaml shape and new-command behavior\n",
		"  template     template placeholders and authoring rules\n",
		"  environment  environment variables, default directories, and precedence\n",
		"  exit-codes   what each exit status means\n",
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
		"invox customer edit",
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

	// No status lists as archived.
	unstatedArchivePath := filepath.Join(archiveDir, "2026-03-05.yaml")
	if err := os.WriteFile(unstatedArchivePath, []byte("customer_id: CUST-NS\ninvoice:\n  number: CUST-NS-001\n  issue_date: 2026-03-05\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(unstatedArchivePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "list"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := strings.Join([]string{
		"2026-03-05.yaml\tCUST-NS\t2026-03-05\tarchived",
		"2026-03-06.yaml\tCUST-YAML\t2026-03-06\tarchived",
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
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

func writeArchivedInvoice(t *testing.T, dir, name, invoiceNumber string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	source := "invoice:\n  number: " + invoiceNumber + "\n"
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
	exitCode := Main(args, factory.New(ios, run.Exec{}, env.System()))
	return exitCode, ios.Out.(*bytes.Buffer).String(), ios.ErrOut.(*bytes.Buffer).String()
}
