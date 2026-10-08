package invoice

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// maxMoneyCents bounds every amount invox computes: 10^13 in the invoice
// currency, held as int64 cents. The bound leaves enough headroom that a sum
// of two bounded amounts cannot overflow int64.
const maxMoneyCents int64 = 1_000_000_000_000_000

// errAmountTooLarge reports an amount above maxMoneyCents. subject names the
// amount, such as "invoice.paid_amount:" or "invoice total".
func errAmountTooLarge(subject string) error {
	return fmt.Errorf("%s exceeds the maximum amount of `%s`", subject, FormatMoneyCents(maxMoneyCents))
}

// moneyCents rounds value to cents like quantizeMoney, and reports false when
// the result lies outside ±maxMoneyCents instead of wrapping around.
func moneyCents(value *big.Rat) (int64, bool) {
	if value == nil {
		return 0, true
	}
	cents := roundHalfUp(new(big.Rat).Mul(value, big.NewRat(100, 1)))
	if cents.CmpAbs(big.NewInt(maxMoneyCents)) > 0 {
		return 0, false
	}
	return cents.Int64(), true
}

// addMoneyCents adds two amounts that are each within ±maxMoneyCents, so the
// int64 sum cannot overflow, and reports false when the sum leaves that range.
func addMoneyCents(left, right int64) (int64, bool) {
	sum := left + right
	if sum > maxMoneyCents || sum < -maxMoneyCents {
		return 0, false
	}
	return sum, true
}

// decimalPattern is the grammar for money, quantities and rates: an optional
// minus sign, digits, and an optional fraction. Leading zeros are decimal.
var decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

func parseDecimal(text string) (*big.Rat, bool) {
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

func percentOfMoney(cents int64, percent *big.Rat) *big.Rat {
	base := new(big.Rat).SetInt64(cents)
	base.Quo(base, big.NewRat(100, 1))
	result := new(big.Rat).Mul(base, percent)
	result.Quo(result, big.NewRat(100, 1))
	return result
}

func quantizeMoney(value *big.Rat) int64 {
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

// FormatMoneyCents formats cents as 1.234,56, without a currency.
func FormatMoneyCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	integerPart := cents / 100
	fractionPart := cents % 100
	return fmt.Sprintf("%s%s,%02d", sign, groupThousands(integerPart), fractionPart)
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

func formatQuantity(value *big.Rat) string {
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
