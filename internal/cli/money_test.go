package cli

import "testing"

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		cents    int64
		currency string
		want     string
	}{
		{12000, "EUR", "120,00 €"},
		{123456, "EUR", "1.234,56 €"},
		{-123456, "EUR", "-1.234,56 €"},
		{5, "EUR", "0,05 €"},
		{12000, "USD", "120,00 USD"},
	}
	for _, tt := range tests {
		if got := formatMoney(tt.cents, tt.currency); got != tt.want {
			t.Errorf("formatMoney(%d, %q) = %q, want %q", tt.cents, tt.currency, got, tt.want)
		}
	}
}
