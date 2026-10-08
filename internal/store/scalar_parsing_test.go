package store

import (
	"os"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

func TestDecodeYAMLKeepsSourceTextOfNumericScalars(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "postal code with leading zero", source: "value: 01067\n", want: "01067"},
		{name: "invoice number with leading zero", source: "value: 0042\n", want: "0042"},
		{name: "hex-looking id", source: "value: 0x10\n", want: "0x10"},
		{name: "octal-looking id", source: "value: 0o17\n", want: "0o17"},
		{name: "phone number", source: "value: 0043123456\n", want: "0043123456"},
		{name: "float keeps trailing zeros", source: "value: 12.50\n", want: "12.50"},
		{name: "exponent", source: "value: 1e3\n", want: "1e3"},
		{name: "plain integer", source: "value: 1010\n", want: "1010"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeForTest[struct {
				Value invoice.Text `yaml:"value"`
			}](t, tt.source).Value
			if got != invoice.Text(tt.want) {
				t.Fatalf("value = %#v, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadContextKeepsLeadingZeroPostalCodeAndInvoiceNumber(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, customersPath, "    postal_code: 1010\n", "    postal_code: 01067\n")
	replaceInFixture(t, invoicePath, "  number: CUST-001-001\n", "  number: 0042\n")

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	if got := ctx.Customer.Address.PostalCode; got != "01067" {
		t.Fatalf("customer postal_code = %#v, want %q", got, "01067")
	}
	if got := ctx.Header.Number; got != "0042" {
		t.Fatalf("invoice number = %q, want %q", got, "0042")
	}
}

func TestLoadContextReadsLeadingZeroAmountsAsDecimal(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, invoicePath, "    unit_price: 100\n    quantity: 2\n", "    unit_price: 0100\n    quantity: 010\n")

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	if got := ctx.LineItems[0].LineTotalCents; got != 100000 {
		t.Fatalf("line total = %d cents, want 100000 (10 x 100.00)", got)
	}
	if got := ctx.SubtotalCents; got != 101000 {
		t.Fatalf("subtotal = %d cents, want 101000", got)
	}
}

func TestLoadContextAmountGrammar(t *testing.T) {
	tests := []struct {
		name      string
		unitPrice string
		wantErr   string
		wantCents int64
	}{
		{name: "hex", unitPrice: "0x10", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `0x10`"},
		{name: "fraction", unitPrice: "1/3", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `1/3`"},
		{name: "leading zero fraction", unitPrice: "010/1", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `010/1`"},
		{name: "exponent", unitPrice: "1e3", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `1e3`"},
		{name: "missing integer part", unitPrice: ".5", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `.5`"},
		{name: "octal prefix", unitPrice: "0o10", wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `0o10`"},
		{name: "quoted fraction", unitPrice: `"1/3"`, wantErr: "positions[1].unit_price: expected a decimal number such as 12 or 12.50, got `1/3`"},
		{name: "negative zero", unitPrice: "-0", wantCents: 0},
		{name: "leading zero", unitPrice: "0100", wantCents: 20000},
		{name: "leading zero decimal", unitPrice: "007.50", wantCents: 1500},
		{name: "quoted leading zero", unitPrice: `"0100"`, wantCents: 20000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			replaceInFixture(t, invoicePath, "    unit_price: 100\n", "    unit_price: "+tt.unitPrice+"\n")

			ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadContext returned nil error, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadContext returned error: %v", err)
			}
			if got := ctx.LineItems[0].LineTotalCents; got != tt.wantCents {
				t.Fatalf("line total = %d cents, want %d", got, tt.wantCents)
			}
		})
	}
}

func TestLoadContextQuantityAndPaidAmountGrammar(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr string
	}{
		{name: "hex quantity", from: "    quantity: 2\n", to: "    quantity: 0x10\n", wantErr: "positions[1].quantity: expected a decimal number such as 12 or 12.50, got `0x10`"},
		{name: "fraction quantity", from: "    quantity: 2\n", to: "    quantity: 1/3\n", wantErr: "positions[1].quantity: expected a decimal number such as 12 or 12.50, got `1/3`"},
		{name: "exponent paid amount", from: "  paid_amount: 0\n", to: "  paid_amount: 1e3\n", wantErr: "invoice.paid_amount: expected a decimal number such as 12 or 12.50, got `1e3`"},
		{name: "negative zero quantity", from: "    quantity: 2\n", to: "    quantity: -0\n", wantErr: "positions[1].quantity: must be > 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			replaceInFixture(t, invoicePath, tt.from, tt.to)

			_, err := LoadContext(customersPath, issuerPath, invoicePath)
			if err == nil {
				t.Fatalf("LoadContext returned nil error, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func replaceInFixture(t *testing.T, path, old, replacement string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	if !strings.Contains(string(source), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(source), old, replacement, 1)), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}
