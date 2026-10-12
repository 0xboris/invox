package cli_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestRootHelpShowsDocumentationTopics(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"-h"})
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
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"help", "config"})
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
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"help", "customers"})
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
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"help", "issuer"})
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
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"help", "defaults"})
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
