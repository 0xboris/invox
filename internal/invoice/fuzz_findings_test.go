package invoice

import (
	"strings"
	"testing"
)

// Regression tests for the #17 bugs that the fuzz targets in fuzz_test.go
// exercise.

func TestLoadContextRejectsAmountsAboveMaximum(t *testing.T) {
	// The fixture has two positions, 100 × 2 and 10 × 1, at 20% VAT.
	tests := []struct {
		name         string
		old, new     string
		wantErr      string
		wantSubtotal int64
	}{
		{
			name:    "unit price",
			old:     "unit_price: 100",
			new:     `unit_price: "100000000000000000"`,
			wantErr: "positions[1].unit_price: exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:    "line total",
			old:     "quantity: 2",
			new:     "quantity: 1000000000000000000000000",
			wantErr: "positions[1]: unit_price × quantity exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:    "subtotal",
			old:     "unit_price: 100",
			new:     "unit_price: 5000000000000",
			wantErr: "invoice subtotal exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:    "VAT amount",
			old:     "vat_percent: 20",
			new:     "vat_percent: 100000000000000000000",
			wantErr: "invoice VAT amount exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:    "total",
			old:     "unit_price: 100",
			new:     "unit_price: 4999999999995",
			wantErr: "invoice total exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:    "paid amount",
			old:     "paid_amount: 0",
			new:     "paid_amount: 1000000000000000000000000000000",
			wantErr: "invoice.paid_amount: exceeds the maximum amount of 10.000.000.000.000,00",
		},
		{
			name:         "subtotal at the maximum",
			old:          "unit_price: 100",
			new:          "unit_price: 4999999999995",
			wantSubtotal: 1_000_000_000_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			replaceInFixture(t, invoicePath, tt.old, tt.new)
			if tt.wantSubtotal != 0 {
				replaceInFixture(t, invoicePath, "vat_percent: 20", "vat_percent: 0")
			}

			ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadContext returned nil error, subtotal %d, total %d", ctx.SubtotalCents, ctx.TotalCents)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadContext returned error: %v", err)
			}
			if ctx.SubtotalCents != tt.wantSubtotal || ctx.TotalCents != tt.wantSubtotal {
				t.Fatalf("subtotal %d, total %d, want %d", ctx.SubtotalCents, ctx.TotalCents, tt.wantSubtotal)
			}
		})
	}
}

func TestIsValidIBANRejectsOutOfRangeCheckDigits(t *testing.T) {
	// Each of these passes the mod-97 check, but ISO 13616 check digits run
	// from 02 to 98.
	for _, iban := range []string{
		"DE00370400440000000060",
		"DE01370400440000000042",
		"DE99370400440000000024",
	} {
		if isValidIBAN(iban) {
			t.Errorf("isValidIBAN(%q) = true, want false", iban)
		}
	}
	if !isValidIBAN("DE89370400440532013000") {
		t.Error(`isValidIBAN("DE89370400440532013000") = false, want true`)
	}
}

func TestParseInvoiceCounterReturnsErrorForInvalidUTF8Pattern(t *testing.T) {
	// regexp.MustCompile panicked on this pattern.
	_, err := parseInvoiceCounter("\xff{customer_id}-{counter}", "\xffCUST-001-1", "CUST-001", "2026-03-06", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("parseInvoiceCounter error = %v, want an invalid UTF-8 error", err)
	}
}

func TestValidateNumberingSettingsRejectsPatternsThatDoNotRoundTrip(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr string
	}{
		{pattern: " {customer_id}-{counter}", wantErr: "must not start or end with whitespace"},
		{pattern: "{customer_id}-{counter}\t", wantErr: "must not start or end with whitespace"},
		{pattern: "{customer_id}-{counter:03}/{counter}", wantErr: "must contain {counter} only once"},
		{pattern: "{customer_code}{counter:03}", wantErr: "needs a separator between {customer_code} and {counter}"},
		{pattern: "{counter}{customer_id}", wantErr: "needs a separator between {customer_id} and {counter}"},
		{pattern: "{customer_id}-{counter:21}", wantErr: "uses an invalid width; use at most 20"},
		{pattern: "\xff{customer_id}-{counter}", wantErr: "must be valid UTF-8"},
		{pattern: defaultNumberingPattern},
		{pattern: "{customer_id}-{year}-{counter:04}"},
		{pattern: "{customer_code}{year}{counter:20}"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := validateNumberingSettings(NumberingSettings{Pattern: tt.pattern, Start: 1})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateNumberingSettings returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateNumberingSettings error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseInvoiceCounterTrimsPatternLikeFormat(t *testing.T) {
	pattern := " {customer_id}-{counter:03} "
	number, err := formatInvoiceNumber(pattern, "CUST-001", nil, "2026-03-06", 7)
	if err != nil {
		t.Fatalf("formatInvoiceNumber returned error: %v", err)
	}
	counter, err := parseInvoiceCounter(pattern, number, "CUST-001", "2026-03-06", nil)
	if err != nil {
		t.Fatalf("parseInvoiceCounter(%q) returned error: %v", number, err)
	}
	if counter != 7 {
		t.Fatalf("parseInvoiceCounter(%q) = %d, want 7", number, counter)
	}
}

func TestBuildEPCPayloadRejectsNonPositiveAmountAndInvalidUTF8(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		epcName string
		wantErr string
	}{
		{name: "zero", cents: 0, epcName: "Boris Consulting", wantErr: "requires an amount above zero"},
		{name: "negative", cents: -500, epcName: "Boris Consulting", wantErr: "requires an amount above zero"},
		{name: "minimum int64", cents: -1 << 63, epcName: "Boris Consulting", wantErr: "requires an amount above zero"},
		{name: "invalid UTF-8", cents: 100, epcName: "\xff", wantErr: "issuer.payment.epc_qr.name: must be valid UTF-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				Currency:         "EUR",
				TotalCents:       tt.cents,
				OutstandingCents: tt.cents,
				IssuerPayment: map[string]any{
					"iban":   "AT611904300234573201",
					"bic":    "BKAUATWW",
					"epc_qr": map[string]any{"name": tt.epcName},
				},
				Invoice: map[string]any{"number": "CUST-001-001"},
			}
			payload, err := buildEPCPayload(ctx)
			if err == nil {
				t.Fatalf("buildEPCPayload returned nil error, payload %q", payload)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
