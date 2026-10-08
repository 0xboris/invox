package money

import (
	"math/big"
	"testing"
)

func TestFormatCents(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{0, "0,00"},
		{5, "0,05"},
		{123456, "1.234,56"},
		{-123456, "-1.234,56"},
		{MaxCents, "10.000.000.000.000,00"},
	}
	for _, tt := range tests {
		if got := FormatCents(tt.cents); got != tt.want {
			t.Errorf("FormatCents(%d) = %q, want %q", tt.cents, got, tt.want)
		}
	}
}

func TestFormatQuantity(t *testing.T) {
	tests := []struct {
		value *big.Rat
		want  string
	}{
		{nil, ""},
		{big.NewRat(3, 1), "3"},
		{big.NewRat(5, 2), "2,5"},
		{big.NewRat(-1, 8), "-0,125"},
	}
	for _, tt := range tests {
		if got := FormatQuantity(tt.value); got != tt.want {
			t.Errorf("FormatQuantity(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestFormatUnitPrice(t *testing.T) {
	tests := []struct {
		value *big.Rat
		want  string
	}{
		{nil, "0,00"},
		{big.NewRat(1234, 1), "1.234,00"},
		{big.NewRat(1, 8), "0,125"},
		{big.NewRat(1, 3), "0,3333"},
		{big.NewRat(-12345, 1000), "-12,345"},
	}
	for _, tt := range tests {
		if got := FormatUnitPrice(tt.value); got != tt.want {
			t.Errorf("FormatUnitPrice(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestAddCentsAndPercentOf(t *testing.T) {
	if got, ok := AddCents(MaxCents, 0); !ok || got != MaxCents {
		t.Errorf("AddCents(MaxCents, 0) = %d, %v, want %d, true", got, ok, MaxCents)
	}
	if _, ok := AddCents(MaxCents, 1); ok {
		t.Error("AddCents(MaxCents, 1) ok = true, want false")
	}
	if got := PercentOf(10000, big.NewRat(20, 1)); got.Cmp(big.NewRat(20, 1)) != 0 {
		t.Errorf("PercentOf(10000, 20) = %s, want 20", got.RatString())
	}
}
