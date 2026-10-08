package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/money"
)

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

func TestLoadContextRejectsLegacyInvoiceAliases(t *testing.T) {
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

	_, err = LoadContext(
		customersPath,
		issuerPath,
		legacyPath,
	)
	if err == nil {
		t.Fatal("LoadContext returned nil error for legacy aliases")
	}
	for _, want := range []string{
		"invoice.period_label: unsupported key; use invoice.period",
		"invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		"line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestLoadContextSupportsPerPositionVATOverrides(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}

	mutated := strings.Replace(string(source), "    quantity: 1\n", "    quantity: 1\n    vat_percent: 10\n", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	if got := len(ctx.VATBreakdowns); got != 2 {
		t.Fatalf("len(VATBreakdowns) = %d, want 2", got)
	}
	if got := money.FormatQuantity(ctx.LineItems[0].VATRatePercent); got != "20" {
		t.Fatalf("LineItems[0].VATRatePercent = %q, want %q", got, "20")
	}
	if got := money.FormatQuantity(ctx.LineItems[1].VATRatePercent); got != "10" {
		t.Fatalf("LineItems[1].VATRatePercent = %q, want %q", got, "10")
	}
	if ctx.SubtotalCents != 21000 {
		t.Fatalf("SubtotalCents = %d, want %d", ctx.SubtotalCents, 21000)
	}
	if ctx.VATAmountCents != 4100 {
		t.Fatalf("VATAmountCents = %d, want %d", ctx.VATAmountCents, 4100)
	}
	if ctx.TotalCents != 25100 {
		t.Fatalf("TotalCents = %d, want %d", ctx.TotalCents, 25100)
	}
	if got := money.FormatQuantity(ctx.VATBreakdowns[0].RatePercent); got != "10" {
		t.Fatalf("VATBreakdowns[0].RatePercent = %q, want %q", got, "10")
	}
	if ctx.VATBreakdowns[0].NetCents != 1000 || ctx.VATBreakdowns[0].VATAmountCents != 100 {
		t.Fatalf("VATBreakdowns[0] = %+v, want net=1000 vat=100", ctx.VATBreakdowns[0])
	}
	if got := money.FormatQuantity(ctx.VATBreakdowns[1].RatePercent); got != "20" {
		t.Fatalf("VATBreakdowns[1].RatePercent = %q, want %q", got, "20")
	}
	if ctx.VATBreakdowns[1].NetCents != 20000 || ctx.VATBreakdowns[1].VATAmountCents != 4000 {
		t.Fatalf("VATBreakdowns[1] = %+v, want net=20000 vat=4000", ctx.VATBreakdowns[1])
	}
}

func TestLoadContextFallsBackToCustomerDefaultVATRate(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)

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
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  vat_percent: 20\n", "", 1)
	if err := os.WriteFile(invoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}

	if got := len(ctx.VATBreakdowns); got != 1 {
		t.Fatalf("len(VATBreakdowns) = %d, want 1", got)
	}
	if got := money.FormatQuantity(ctx.LineItems[0].VATRatePercent); got != "13" {
		t.Fatalf("LineItems[0].VATRatePercent = %q, want %q", got, "13")
	}
	if ctx.VATAmountCents != 2730 {
		t.Fatalf("VATAmountCents = %d, want %d", ctx.VATAmountCents, 2730)
	}
	if ctx.TotalCents != 23730 {
		t.Fatalf("TotalCents = %d, want %d", ctx.TotalCents, 23730)
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
