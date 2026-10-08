package money

import (
	"math/big"
	"strings"
	"testing"
)

func FuzzParseDecimal(f *testing.F) {
	for _, seed := range []string{
		"12", "12.50", "-3.5", " 7 ", "010", "-0", "0.005", "-0.005", "0.125",
		"0x10", "0o10", "0b10", "1/3", "010/1", "1e3", ".5", "5.", "+5", "1_000", "1,5", "",
		// #17: amounts that overflowed int64 cents.
		"100000000000000000", "1000000000000000000000000000000",
		"10000000000000", "10000000000000.004", "10000000000000.005", "-10000000000000.005",
	} {
		f.Add(seed)
	}

	maxCents := big.NewInt(MaxCents)
	f.Fuzz(func(t *testing.T, text string) {
		value, ok := ParseDecimal(text)
		want, wantOK := decimalOracle(strings.TrimSpace(text))
		if ok != wantOK {
			t.Fatalf("ParseDecimal(%q) ok = %v, want %v", text, ok, wantOK)
		}
		if !ok {
			return
		}
		if value.Cmp(want) != 0 {
			t.Fatalf("ParseDecimal(%q) = %s, want %s", text, value.RatString(), want.RatString())
		}

		// Accepted values never overflow downstream: they either fit in the
		// money range, rounded exactly, or are reported as too large.
		wantCents := centsOracle(strings.TrimSpace(text))
		cents, ok := Cents(value)
		if fits := wantCents.CmpAbs(maxCents) <= 0; ok != fits {
			t.Fatalf("Cents(%q) ok = %v, want %v (cents %s)", text, ok, fits, wantCents)
		}
		if !ok {
			return
		}
		if cents != wantCents.Int64() {
			t.Fatalf("Cents(%q) = %d, want %s", text, cents, wantCents)
		}
		if quantized := quantize(value); quantized != cents {
			t.Fatalf("quantize(%q) = %d, want %d", text, quantized, cents)
		}
	})
}

// decimalOracle accepts `^-?[0-9]+(\.[0-9]+)?$` with a hand-written scanner,
// so it does not share code with ParseDecimal.
func decimalOracle(text string) (*big.Rat, bool) {
	digits := strings.TrimPrefix(text, "-")
	whole, fraction, hasPoint := strings.Cut(digits, ".")
	if whole == "" || (hasPoint && fraction == "") {
		return nil, false
	}
	for _, r := range whole + fraction {
		if r < '0' || r > '9' {
			return nil, false
		}
	}
	numerator, _ := new(big.Int).SetString(whole+fraction, 10)
	if digits != text {
		numerator.Neg(numerator)
	}
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fraction))), nil)
	return new(big.Rat).SetFrac(numerator, denominator), true
}

// centsOracle rounds an accepted decimal to cents, halves away from zero, by
// looking at its digits.
func centsOracle(text string) *big.Int {
	digits := strings.TrimPrefix(text, "-")
	whole, fraction, _ := strings.Cut(digits, ".")
	fraction += "000"
	cents, _ := new(big.Int).SetString(whole+fraction[:2], 10)
	if fraction[2] >= '5' {
		cents.Add(cents, big.NewInt(1))
	}
	if digits != text {
		cents.Neg(cents)
	}
	return cents
}
