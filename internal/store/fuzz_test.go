package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0xboris/invox/internal/epc"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/money"
)

// The fuzz targets below run their seeds, and every crasher committed under
// testdata/fuzz/<Target>/, as part of `go test`. Run one for longer with
//
//	go test -run='^$' -fuzz='^FuzzLoadContext$' -fuzztime=60s ./internal/store
//
// or all of them with `make fuzz`.

func FuzzBuildEPCPayload(f *testing.F) {
	for _, seed := range []struct {
		name, iban, bic, ref string
		cents                int64
	}{
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "CUST-001-001", 25200},
		{"Boris Consulting", "AT61 1904 3002 3457 3201", "", "", 1},
		{"Zürich Café GmbH", "GI75NWBK000000007099453", "NWBKGI2G", "Rechnung №1", 99999999999},
		{strings.Repeat("N", 70), "AT611904300234573201", "BKAUATWW", strings.Repeat("R", 140), 100},
		{"Boris Consulting", "DE01370400440000000042", "", "", 100},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "line\nbreak", 100},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", 0},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", -500},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", -9223372036854775808},
		{"\xff", "AT611904300234573201", "BKAUATWW", "", 100},
	} {
		f.Add(seed.name, seed.iban, seed.bic, seed.ref, seed.cents)
	}

	f.Fuzz(func(t *testing.T, name, iban, bic, ref string, cents int64) {
		ctx := &invoice.Context{
			Currency:         "EUR",
			TotalCents:       cents,
			OutstandingCents: cents,
			Company:          invoice.Company{LegalCompanyName: "Fallback Name"},
			Payment: invoice.Payment{
				IBAN:  invoice.Text(iban),
				BIC:   invoice.Text(bic),
				EPCQR: invoice.EPCQR{Name: invoice.Text(name), Text: invoice.Text(ref)},
			},
			Header: invoice.Header{Number: "CUST-001-001"},
		}
		payload, err := buildEPCPayload(ctx)
		if err != nil {
			return
		}
		if len(payload) > epc.MaxPayloadBytes {
			t.Fatalf("payload is %d bytes, more than %d: %q", len(payload), epc.MaxPayloadBytes, payload)
		}
		if !utf8.Valid(payload) {
			t.Fatalf("payload is not valid UTF-8: %q", payload)
		}
		lines := strings.Split(string(payload), "\n")
		if len(lines) < 8 || len(lines) > 12 || lines[0] != "BCD" || lines[3] != "SCT" {
			t.Fatalf("payload does not have the EPC layout: %q", payload)
		}
		if !epc.ValidIBAN(lines[6]) {
			t.Fatalf("payload carries invalid IBAN %q", lines[6])
		}
		if cents < 1 || cents > epc.MaxAmountCents {
			t.Fatalf("payload accepted amount %d cents outside 0.01-999999999.99: %q", cents, payload)
		}
		if want := fmt.Sprintf("EUR%d.%02d", cents/100, cents%100); lines[7] != want {
			t.Fatalf("payload amount line = %q, want %q", lines[7], want)
		}
	})
}

func FuzzLoadContext(f *testing.F) {
	for _, seed := range []string{
		fuzzInvoiceYAML,
		strings.Replace(fuzzInvoiceYAML, "paid_amount: 0", "paid_amount: 252", 1),
		strings.Replace(fuzzInvoiceYAML, "vat_percent: 20", "vat_percent: 0.125", 1),
		// #17: amounts that overflowed int64 cents.
		strings.Replace(fuzzInvoiceYAML, "unit_price: 100", `unit_price: "100000000000000000"`, 1),
		strings.Replace(fuzzInvoiceYAML, "quantity: 2", "quantity: 1000000000000000000000000", 1),
		strings.Replace(fuzzInvoiceYAML, "paid_amount: 0", "paid_amount: 1000000000000000000000000000000", 1),
		strings.Replace(fuzzInvoiceYAML, "vat_percent: 20", "vat_percent: 100000000000000000000", 1),
		"",
		"[]",
		strings.Replace(strings.Replace(fuzzInvoiceYAML, "  - name: Development", "  - &dev\n    name: Development", 1),
			"  - name: Support\n    description: QA\n    unit_price: 10\n    quantity: 1\n", "  - *dev\n", 1),
		// #17: an alias to its enclosing node, and a billion laughs.
		"customer_id: CUST-001\npositions: &p [*p]\n",
		billionLaughs(),
	} {
		f.Add([]byte(seed))
	}

	dir := f.TempDir()
	customersPath, issuerPath := writeFuzzCustomerAndIssuer(f, dir)
	f.Fuzz(func(t *testing.T, source []byte) {
		invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
		if err := os.WriteFile(invoicePath, source, 0o644); err != nil {
			t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
		}
		ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
		if err != nil {
			return
		}
		checkContextTotals(t, ctx)
	})
}

// checkContextTotals asserts the money invariants of a loaded invoice.
func checkContextTotals(t *testing.T, ctx *invoice.Context) {
	t.Helper()

	inRange := func(label string, cents int64) {
		if cents < 0 || cents > money.MaxCents {
			t.Fatalf("%s = %d cents, outside 0..%d", label, cents, money.MaxCents)
		}
	}
	if len(ctx.LineItems) == 0 {
		t.Fatal("loaded an invoice without line items")
	}
	var subtotal int64
	for index, item := range ctx.LineItems {
		inRange("line total", item.LineTotalCents)
		if _, ok := money.Cents(item.UnitPrice); !ok {
			t.Fatalf("positions[%d].unit_price %s does not fit in cents", index+1, item.UnitPrice.RatString())
		}
		subtotal += item.LineTotalCents
	}
	inRange("subtotal", ctx.SubtotalCents)
	if subtotal != ctx.SubtotalCents {
		t.Fatalf("line totals add up to %d, subtotal is %d", subtotal, ctx.SubtotalCents)
	}
	var net, vat int64
	for _, breakdown := range ctx.VATBreakdowns {
		inRange("VAT amount", breakdown.VATAmountCents)
		net += breakdown.NetCents
		vat += breakdown.VATAmountCents
	}
	if net != ctx.SubtotalCents || vat != ctx.VATAmountCents {
		t.Fatalf("VAT breakdowns add up to net %d and VAT %d, want %d and %d", net, vat, ctx.SubtotalCents, ctx.VATAmountCents)
	}
	inRange("total", ctx.TotalCents)
	if ctx.TotalCents != ctx.SubtotalCents+ctx.VATAmountCents {
		t.Fatalf("total %d != subtotal %d + VAT %d", ctx.TotalCents, ctx.SubtotalCents, ctx.VATAmountCents)
	}
	inRange("paid amount", ctx.PaidAmountCents)
	inRange("outstanding amount", ctx.OutstandingCents)
	if ctx.OutstandingCents != ctx.TotalCents-ctx.PaidAmountCents {
		t.Fatalf("outstanding %d != total %d - paid %d", ctx.OutstandingCents, ctx.TotalCents, ctx.PaidAmountCents)
	}
}

const fuzzInvoiceYAML = `customer_id: CUST-001
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
`

// writeFuzzCustomerAndIssuer writes the customers.yaml and issuer.yaml that
// writeContextFixtures uses, for targets that cannot take a *testing.T.
func writeFuzzCustomerAndIssuer(tb testing.TB, dir string) (string, string) {
	tb.Helper()

	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	files := map[string]string{
		customersPath: `CUST-001:
  name: Appsters GmbH
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
`,
		issuerPath: `company:
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
`,
	}
	for path, source := range files {
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			tb.Fatalf("WriteFile(%s) returned error: %v", filepath.Base(path), err)
		}
	}
	return customersPath, issuerPath
}
