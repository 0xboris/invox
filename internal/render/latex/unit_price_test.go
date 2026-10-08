package latex

import (
	"testing"

	"github.com/0xboris/invox/internal/money"
)

func TestFormatUnitPriceShowsNeededDecimals(t *testing.T) {
	tests := []struct {
		price    string
		currency string
		want     string
	}{
		{price: "100", currency: "EUR", want: `100,00 \euro`},
		{price: "1234.5", currency: "EUR", want: `1.234,50 \euro`},
		{price: "0.10000", currency: "EUR", want: `0,10 \euro`},
		{price: "0.125", currency: "EUR", want: `0,125 \euro`},
		{price: "0.1234", currency: "USD", want: "0,1234 USD"},
		{price: "1234.56789", currency: "EUR", want: `1.234,5679 \euro`},
		{price: "0.12345", currency: "EUR", want: `0,1235 \euro`},
		{price: "0.00001", currency: "EUR", want: `0,0000 \euro`},
	}
	for _, tt := range tests {
		price, ok := money.ParseDecimal(tt.price)
		if !ok {
			t.Fatalf("ParseDecimal(%q) failed", tt.price)
		}
		if got := formatUnitPrice(price, tt.currency); got != tt.want {
			t.Errorf("formatUnitPrice(%s, %s) = %q, want %q", tt.price, tt.currency, got, tt.want)
		}
	}
}
