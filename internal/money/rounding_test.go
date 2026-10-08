package money

import (
	"math/big"
	"testing"
)

func TestCentsRoundsHalvesAwayFromZero(t *testing.T) {
	tests := []struct {
		value *big.Rat
		want  string
	}{
		{big.NewRat(5, 1000), "0,01"},
		{big.NewRat(-5, 1000), "-0,01"},
		{big.NewRat(4, 1000), "0,00"},
		{big.NewRat(15, 1000), "0,02"},
		{big.NewRat(25, 1000), "0,03"},
	}
	for _, tt := range tests {
		cents, ok := Cents(tt.value)
		if !ok {
			t.Fatalf("Cents(%s) ok = false", tt.value.RatString())
		}
		if got := FormatCents(cents); got != tt.want {
			t.Errorf("FormatCents(Cents(%s)) = %q, want %q", tt.value.RatString(), got, tt.want)
		}
	}
}
