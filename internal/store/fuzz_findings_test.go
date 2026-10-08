package store

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

// Regression tests for the #17 bugs that the fuzz targets in fuzz_test.go
// exercise.

func TestLoadContextRejectsAmountsAboveMaximum(t *testing.T) {
	// The fixture has two positions, 100 × 2 and 10 × 1, at 20% VAT.
	tests := []struct {
		name         string
		old, new     string
		wantErr      string
		wantNoErr    string
		wantSubtotal int64
	}{
		{
			name:    "unit price",
			old:     "unit_price: 100",
			new:     `unit_price: "100000000000000000"`,
			wantErr: "positions[1].unit_price: exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:    "line total",
			old:     "quantity: 2",
			new:     "quantity: 1000000000000000000000000",
			wantErr: "positions[1]: unit_price × quantity exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:    "subtotal",
			old:     "unit_price: 100",
			new:     "unit_price: 5000000000000",
			wantErr: "invoice subtotal exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:    "VAT amount",
			old:     "vat_percent: 20",
			new:     "vat_percent: 100000000000000000000",
			wantErr: "invoice VAT amount exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:    "total",
			old:     "unit_price: 100",
			new:     "unit_price: 4999999999995",
			wantErr: "invoice total exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:    "paid amount",
			old:     "paid_amount: 0",
			new:     "paid_amount: 1000000000000000000000000000000",
			wantErr: "invoice.paid_amount: exceeds the maximum amount of `10.000.000.000.000,00`",
		},
		{
			name:      "huge negative unit price",
			old:       "unit_price: 100",
			new:       "unit_price: -100000000000000000",
			wantErr:   "positions[1].unit_price: must be >= 0",
			wantNoErr: "exceeds the maximum",
		},
		{
			name:      "huge negative paid amount",
			old:       "paid_amount: 0",
			new:       "paid_amount: -1000000000000000000000000000000",
			wantErr:   "invoice.paid_amount: must not be negative",
			wantNoErr: "exceeds the maximum",
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
				if tt.wantNoErr != "" && strings.Contains(err.Error(), tt.wantNoErr) {
					t.Fatalf("error %q also contains %q", err.Error(), tt.wantNoErr)
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
			ctx := &invoice.Context{
				Currency:         "EUR",
				TotalCents:       tt.cents,
				OutstandingCents: tt.cents,
				Payment: invoice.Payment{
					IBAN:  "AT611904300234573201",
					BIC:   "BKAUATWW",
					EPCQR: invoice.EPCQR{Name: invoice.Text(tt.epcName)},
				},
				Header: invoice.Header{Number: "CUST-001-001"},
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
