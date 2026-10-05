package invoice

import (
	"fmt"
	"math/big"
)

// maxMoneyCents bounds every amount invox computes: 10^13 in the invoice
// currency, held as int64 cents. The bound leaves enough headroom that a sum
// of two bounded amounts cannot overflow int64.
const maxMoneyCents int64 = 1_000_000_000_000_000

// errAmountTooLarge reports an amount above maxMoneyCents. subject names the
// amount, such as "invoice.paid_amount:" or "invoice total".
func errAmountTooLarge(subject string) error {
	return fmt.Errorf("%s exceeds the maximum amount of `%s`", subject, formatMoneyCents(maxMoneyCents))
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
