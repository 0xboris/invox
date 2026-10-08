package invoice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #17: customer_id: 1001 is the ID "1001", not a missing value.
func TestLoadContextReadsNumericCustomerID(t *testing.T) {
	for _, id := range []string{"1001", "0042"} {
		t.Run(id, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			replaceInFixture(t, customersPath, "CUST-001:\n", id+":\n")
			replaceInFixture(t, invoicePath, "customer_id: CUST-001\n", "customer_id: "+id+"\n")

			ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
			if err != nil {
				t.Fatalf("LoadContext returned error: %v", err)
			}
			if ctx.CustomerID != id {
				t.Fatalf("CustomerID = %q, want %q", ctx.CustomerID, id)
			}
			if got := ctx.Customer.DisplayName(); got != "Appsters GmbH" {
				t.Fatalf("customer name = %q, want %q", got, "Appsters GmbH")
			}
		})
	}
}

// #17: a value of the wrong kind is an error with file:line and the field,
// never its Go string such as map[first:Appsters].
func TestLoadContextRejectsValuesOfTheWrongKind(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		from    string
		to      string
		wantErr string
	}{
		{name: "customer name mapping", file: "customers", from: "  name: Appsters GmbH\n", to: "  name: {first: Appsters}\n", wantErr: "%s:2: CUST-001.name: expected a string, got a mapping"},
		{name: "customer street list", file: "customers", from: "    street: Hauptstrasse 1\n", to: "    street: [Hauptstrasse, 1]\n", wantErr: "%s:8: CUST-001.address.street: expected a string, got a list"},
		{name: "customer address scalar", file: "customers", from: "  address:\n    street: Hauptstrasse 1\n    postal_code: 1010\n    city: Vienna\n    country: Austria\n", to: "  address: Hauptstrasse 1\n", wantErr: "%s:7: CUST-001.address must be a mapping, got a string"},
		{name: "issuer website mapping", file: "issuer", from: "  website: https://example.com\n", to: "  website: {url: https://example.com}\n", wantErr: "%s:5: company.website: expected a string, got a mapping"},
		{name: "issuer due_days list", file: "issuer", from: "  due_days: 30\n", to: "  due_days: [30]\n", wantErr: "%s:16: payment.due_days: expected an integer, got a list"},
		{name: "invoice period list", file: "invoice", from: "  period: Leistungszeitraum\n", to: "  period: [March]\n", wantErr: "%s:6: invoice.period: expected a string, got a list"},
		{name: "invoice date mapping", file: "invoice", from: "  issue_date: 2026-03-06\n", to: "  issue_date: {day: 6}\n", wantErr: "%s:4: invoice.issue_date: expected YYYY-MM-DD, got a mapping"},
		{name: "position name mapping", file: "invoice", from: "  - name: Support\n", to: "  - name: {de: Support}\n", wantErr: "%s:14: positions[2].name: expected a string, got a mapping"},
		{name: "position price list", file: "invoice", from: "    unit_price: 10\n", to: "    unit_price: [10]\n", wantErr: "%s:16: positions[2].unit_price: expected a decimal number such as 12 or 12.50, got a list"},
		{name: "position vat mapping", file: "invoice", from: "    unit_price: 10\n", to: "    unit_price: 10\n    vat_percent: {rate: 10}\n", wantErr: "%s:17: positions[2].vat_percent: expected a number or percent string, got a mapping"},
		{name: "positions mapping", file: "invoice", from: "positions:\n", to: "positions: {}\nunused:\n", wantErr: "%s:9: positions must be a list, got a mapping"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			path := map[string]string{"customers": customersPath, "issuer": issuerPath, "invoice": invoicePath}[tt.file]
			replaceInFixture(t, path, tt.from, tt.to)

			_, err := LoadContext(customersPath, issuerPath, invoicePath)
			want := fmt.Sprintf(tt.wantErr, path)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to contain %q", err, want)
			}
			if strings.Contains(err.Error(), "map[") {
				t.Fatalf("error %q contains a Go map string", err)
			}
			var decodeErr *DecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("error %v is not a *DecodeError", err)
			}
		})
	}
}

func TestLoadContextRejectsUnknownKeys(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		from    string
		to      string
		wantErr string
	}{
		{name: "customer", file: "customers", from: "  status: active\n", to: "  status: active\n  phone: 123\n", wantErr: `%s:4: unknown key "phone" in CUST-001`},
		{name: "customer address", file: "customers", from: "    city: Vienna\n", to: "    city: Vienna\n    floor: 3\n", wantErr: `%s:11: unknown key "floor" in CUST-001.address`},
		{name: "issuer top level", file: "issuer", from: "payment:\n", to: "bank: x\npayment:\n", wantErr: `%s:12: unknown key "bank"`},
		{name: "issuer payment", file: "issuer", from: "  bic: BKAUATWW\n", to: "  bic: BKAUATWW\n  swift: BKAUATWW\n", wantErr: `%s:16: unknown key "swift" in payment`},
		{name: "invoice header", file: "invoice", from: "  period: Leistungszeitraum\n", to: "  period: Leistungszeitraum\n  currency: EUR\n", wantErr: `%s:7: unknown key "currency" in invoice`},
		{name: "position", file: "invoice", from: "    quantity: 2\n", to: "    quantity: 2\n    unit: h\n", wantErr: `%s:14: unknown key "unit" in positions[1]`},
		{name: "anchor holder", file: "invoice", from: "customer_id: CUST-001\n", to: "customer_id: CUST-001\nextra: &extra {unit: h}\n", wantErr: ""},
		{name: "removed line_items", file: "invoice", from: "positions:\n", to: "line_items: []\npositions:\n", wantErr: "%s:9: line_items: unsupported key; use positions"},
		{name: "removed period_label", file: "invoice", from: "  period: Leistungszeitraum\n", to: "  period: Leistungszeitraum\n  period_label: March\n", wantErr: "%s:7: invoice.period_label: unsupported key; use invoice.period"},
		{name: "removed vat_rate_percent", file: "invoice", from: "  vat_percent: 20\n", to: "  vat_percent: 20\n  vat_rate_percent: 20\n", wantErr: "%s:8: invoice.vat_rate_percent: unsupported key; use invoice.vat_percent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			path := map[string]string{"customers": customersPath, "issuer": issuerPath, "invoice": invoicePath}[tt.file]
			replaceInFixture(t, path, tt.from, tt.to)

			_, err := LoadContext(customersPath, issuerPath, invoicePath)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("LoadContext returned error: %v", err)
				}
				return
			}
			if want := fmt.Sprintf(tt.wantErr, path); err == nil || err.Error() != want {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestLoadContextRejectsAKeyMergedIntoAPosition(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, invoicePath, "customer_id: CUST-001\n", "customer_id: CUST-001\nextra: &extra {unit: h}\n")
	replaceInFixture(t, invoicePath, "  - name: Support\n", "  - <<: *extra\n    name: Support\n")

	_, err := LoadContext(customersPath, issuerPath, invoicePath)
	if want := invoicePath + `:2: unknown key "unit" in positions[2]`; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestLoadContextReportsEveryDecodeProblemInFileOrder(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, customersPath, "  status: active\n", "  status: [active]\n")
	replaceInFixture(t, issuerPath, "  due_days: 30\n", "  due_days: soon\n")
	replaceInFixture(t, invoicePath, "  issue_date: 2026-03-06\n", "  issue_date: 06.03.2026\n")
	replaceInFixture(t, invoicePath, "    unit_price: 10\n", "    unit_price: ten\n")

	_, err := LoadContext(customersPath, issuerPath, invoicePath)
	want := strings.Join([]string{
		customersPath + ":3: CUST-001.status: expected a string, got a list",
		issuerPath + ":16: payment.due_days: expected an integer, got `soon`",
		invoicePath + ":4: invoice.issue_date: expected YYYY-MM-DD, got `06.03.2026`",
		invoicePath + ":16: positions[2].unit_price: expected a decimal number such as 12 or 12.50, got `ten`",
	}, "\n")
	if err == nil || err.Error() != want {
		t.Fatalf("error =\n%v\nwant\n%s", err, want)
	}
}

// Only the invoice's own customer is decoded, so another entry of
// customers.yaml, such as a holder of anchors, cannot stop the invoice.
func TestLoadContextDecodesOnlyTheInvoicesCustomer(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	source, err := os.ReadFile(customersPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := "_defaults: &defaults\n  street: Ring 1\nOTHER:\n  name: {first: Other}\n"
	if err := os.WriteFile(customersPath, append([]byte(broken), source...), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadContext(customersPath, issuerPath, invoicePath); err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
}

func TestListCustomersIgnoresUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "customers.yaml")
	source := "B:\n  legal_company_name: Bee KG\n  phone: 123\nA:\n  name: Ay GmbH\n  status: active\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	customers, err := ListCustomers(path)
	if err != nil {
		t.Fatalf("ListCustomers returned error: %v", err)
	}
	got := fmt.Sprint(customers)
	if want := "[{A Ay GmbH active} {B Bee KG }]"; got != want {
		t.Fatalf("customers = %s, want %s", got, want)
	}
}

func TestListCustomersRejectsAValueOfTheWrongKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "customers.yaml")
	if err := os.WriteFile(path, []byte("A:\n  name: [Ay, GmbH]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ListCustomers(path)
	if want := path + ":2: A.name: expected a string, got a list"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
