// Package money parses decimal amounts and does the cent arithmetic and
// formatting for invoices: half-up rounding and the 1.234,56 form.
package money

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// MaxCents bounds every amount invox computes: 10^13 in the invoice
// currency, held as int64 cents. The bound leaves enough headroom that a sum
// of two bounded amounts cannot overflow int64.
const MaxCents int64 = 1_000_000_000_000_000

// Cents rounds value to cents like quantize, and reports false when
// the result lies outside ±MaxCents instead of wrapping around.
func Cents(value *big.Rat) (int64, bool) {
	if value == nil {
		return 0, true
	}
	cents := roundHalfUp(new(big.Rat).Mul(value, big.NewRat(100, 1)))
	if cents.CmpAbs(big.NewInt(MaxCents)) > 0 {
		return 0, false
	}
	return cents.Int64(), true
}

// AddCents adds two amounts that are each within ±MaxCents, so the
// int64 sum cannot overflow, and reports false when the sum leaves that range.
func AddCents(left, right int64) (int64, bool) {
	sum := left + right
	if sum > MaxCents || sum < -MaxCents {
		return 0, false
	}
	return sum, true
}

// decimalPattern is the grammar for money, quantities and rates: an optional
// minus sign, digits, and an optional fraction. Leading zeros are decimal.
var decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// ParseDecimal reads text in the decimalPattern grammar, ignoring surrounding
// space, and reports false for anything else.
func ParseDecimal(text string) (*big.Rat, bool) {
	text = strings.TrimSpace(text)
	if !decimalPattern.MatchString(text) {
		return nil, false
	}
	rat := new(big.Rat)
	if _, ok := rat.SetString(text); ok {
		return rat, true
	}
	return nil, false
}

// PercentOf returns percent per cent of cents in currency units, unrounded.
func PercentOf(cents int64, percent *big.Rat) *big.Rat {
	base := new(big.Rat).SetInt64(cents)
	base.Quo(base, big.NewRat(100, 1))
	result := new(big.Rat).Mul(base, percent)
	result.Quo(result, big.NewRat(100, 1))
	return result
}

func quantize(value *big.Rat) int64 {
	if value == nil {
		return 0
	}
	scaled := new(big.Rat).Mul(value, big.NewRat(100, 1))
	return roundHalfUpToInt(scaled)
}

func roundHalfUpToInt(value *big.Rat) int64 {
	if value == nil {
		return 0
	}
	return roundHalfUp(value).Int64()
}

// roundHalfUp rounds value to the nearest integer, with halves away from zero.
func roundHalfUp(value *big.Rat) *big.Int {
	numerator := new(big.Int).Set(value.Num())
	denominator := new(big.Int).Set(value.Denom())
	sign := numerator.Sign()
	if sign == 0 {
		return numerator
	}
	if sign < 0 {
		numerator.Neg(numerator)
	}
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	twiceRemainder := new(big.Int).Lsh(remainder, 1)
	if twiceRemainder.Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if sign < 0 {
		quotient.Neg(quotient)
	}
	return quotient
}

// FormatCents formats cents as 1.234,56, without a currency.
func FormatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	integerPart := cents / 100
	fractionPart := cents % 100
	return fmt.Sprintf("%s%s,%02d", sign, groupThousands(integerPart), fractionPart)
}

// DecimalString formats cents as a plain decimal with a point and no
// grouping, such as 1234.56, the form machine output uses.
func DecimalString(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func groupThousands(value int64) string {
	digits := fmt.Sprintf("%d", value)
	if len(digits) <= 3 {
		return digits
	}
	parts := make([]string, 0, (len(digits)+2)/3)
	for len(digits) > 3 {
		parts = append(parts, digits[len(digits)-3:])
		digits = digits[:len(digits)-3]
	}
	parts = append(parts, digits)
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, ".")
}

// FormatQuantity formats a quantity or rate with a decimal comma and no
// trailing zeros, such as 2,5.
func FormatQuantity(value *big.Rat) string {
	if value == nil {
		return ""
	}
	if value.Denom().Cmp(big.NewInt(1)) == 0 {
		return value.Num().String()
	}
	text := value.FloatString(10)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return strings.ReplaceAll(text, ".", ",")
}

// Unit prices are shown with as many decimals as they need, at least
// minUnitPriceDecimals and at most maxUnitPriceDecimals (rounded half up beyond
// that), so that unit price × quantity matches the line total.
const (
	minUnitPriceDecimals = 2
	maxUnitPriceDecimals = 4
)

// FormatUnitPrice formats a unit price like FormatCents, with as
// many decimals as it needs.
func FormatUnitPrice(value *big.Rat) string {
	decimals := unitPriceDecimals(value)
	if decimals == minUnitPriceDecimals {
		return FormatCents(quantize(value))
	}
	scale := int64(1)
	for range decimals {
		scale *= 10
	}
	units := roundHalfUpToInt(new(big.Rat).Mul(value, new(big.Rat).SetInt64(scale)))
	sign := ""
	if units < 0 {
		sign = "-"
		units = -units
	}
	return fmt.Sprintf("%s%s,%0*d", sign, groupThousands(units/scale), decimals, units%scale)
}

func unitPriceDecimals(value *big.Rat) int {
	if value == nil {
		return minUnitPriceDecimals
	}
	scaled := new(big.Rat).Set(value)
	scaled.Mul(scaled, big.NewRat(100, 1))
	for decimals := minUnitPriceDecimals; decimals < maxUnitPriceDecimals; decimals++ {
		if scaled.IsInt() {
			return decimals
		}
		scaled.Mul(scaled, big.NewRat(10, 1))
	}
	return maxUnitPriceDecimals
}
