package epc

import (
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func FuzzIsValidIBAN(f *testing.F) {
	for _, seed := range []string{
		"AT611904300234573201",
		"PL61109010140000071219812874",
		"DE89370400440532013000",
		"GI75NWBK000000007099453",
		"ZZ6600000000000",
		"AT61190430023457320",
		"ATAA1904300234573201",
		"at611904300234573201",
		// #17: check digits 00, 01 and 99 never occur in a valid IBAN.
		"DE01370400440000000042",
		"DE00370400440000000",
		"DE99370400440000000000",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		got := ValidIBAN(value)
		if want := ibanOracle(value); got != want {
			t.Fatalf("ValidIBAN(%q) = %v, want %v", value, got, want)
		}
	})
}

// ibanOracle checks an IBAN with big-integer arithmetic: a known country and
// length, upper-case letters and digits only, check digits 02-98, and the
// rearranged number mod 97 == 1.
func ibanOracle(value string) bool {
	if len(value) < 4 || ibanCountryLengths[value[:2]] != len(value) {
		return false
	}
	checkDigits := value[2:4]
	if checkDigits[0] < '0' || checkDigits[0] > '9' || checkDigits[1] < '0' || checkDigits[1] > '9' {
		return false
	}
	if checkDigits < "02" || checkDigits > "98" {
		return false
	}
	var numeric strings.Builder
	for _, r := range value[4:] + value[:4] {
		switch {
		case r >= '0' && r <= '9':
			numeric.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			numeric.WriteString(strconv.Itoa(int(r-'A') + 10))
		default:
			return false
		}
	}
	number, ok := new(big.Int).SetString(numeric.String(), 10)
	if !ok {
		return false
	}
	return new(big.Int).Mod(number, big.NewInt(97)).Int64() == 1
}
