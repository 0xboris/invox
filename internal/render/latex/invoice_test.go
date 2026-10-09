package latex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/render/latex"
)

// host is a user's config home and home directory.
type host struct {
	configHome string
	home       string
}

func testHost(configHome, home string) host {
	return host{configHome: configHome, home: home}
}

// isolatedHost returns a host whose directories are all under a fresh
// temporary directory, so a test never reads the developer's config.
func isolatedHost(t *testing.T) host {
	t.Helper()
	root := t.TempDir()
	return testHost(filepath.Join(root, "config-home"), filepath.Join(root, "home"))
}

// service returns the use cases as invox wires them for h, with the
// support files named as on a command line.
func (h host) service(t *testing.T, files cmdutil.Files) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home: h.home,
		Vars: map[string]string{"XDG_CONFIG_HOME": h.configHome},
	}).Service(files)
}

// loadContext loads the invoice at invoicePath with its customer and
// issuer, as validate does.
func loadContext(t *testing.T, customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	t.Helper()
	result, err := isolatedHost(t).service(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}).Validate(invoicePath)
	return result.Context, err
}

// template resolves the template at path as render and build do.
func (h host) template(t *testing.T, path string) billing.Template {
	t.Helper()
	tmpl, err := h.service(t, cmdutil.Files{}).Directory.Template(path)
	if err != nil {
		t.Fatalf("Template(%s) returned error: %v", path, err)
	}
	return tmpl
}

// renderInvoice fills the template at templatePath with ctx and writes it,
// with the assets it uses, to outputPath, as render does.
func (h host) renderInvoice(t *testing.T, templatePath, outputPath string, ctx *invoice.Context) error {
	t.Helper()
	tmpl := h.template(t, templatePath)
	source, err := latex.Renderer{}.Render(tmpl, ctx, billing.EPCFor(ctx))
	if err != nil {
		return err
	}
	return latex.Renderer{}.Write(tmpl, source, outputPath)
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

	if err := os.WriteFile(logoPath, []byte("logo"), 0o644); err != nil {
		t.Fatalf("WriteFile(logo.png) returned error: %v", err)
	}
	if err := os.WriteFile(fontPath, []byte("font"), 0o644); err != nil {
		t.Fatalf("WriteFile(font) returned error: %v", err)
	}

	return customersPath, issuerPath, invoicePath, templatePath, logoPath, fontPath
}
