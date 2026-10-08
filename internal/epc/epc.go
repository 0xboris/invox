// Package epc holds the rules of the EPC069-12 QR code for SEPA credit
// transfers: IBAN, BIC, purpose and text checks, and the payload layout.
package epc

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxPayloadBytes = 331
	MaxNameChars    = 70
	maxPurposeChars = 4
	MaxTextChars    = 140
	MaxInfoChars    = 70
	MaxAmountCents  = 99999999999
)

var (
	purposePattern     = regexp.MustCompile(`^[A-Za-z0-9]{1,4}$`)
	bicPattern         = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
	ibanCountryLengths = map[string]int{
		"AD": 24,
		"AE": 23,
		"AL": 28,
		"AT": 20,
		"AZ": 28,
		"BA": 20,
		"BE": 16,
		"BG": 22,
		"BH": 22,
		"BI": 16,
		"BR": 29,
		"BY": 28,
		"CH": 21,
		"CR": 22,
		"CY": 28,
		"CZ": 24,
		"DE": 22,
		"DJ": 27,
		"DK": 18,
		"DO": 28,
		"EE": 20,
		"EG": 29,
		"ES": 24,
		"FI": 18,
		"FK": 18,
		"FO": 18,
		"FR": 27,
		"GB": 22,
		"GE": 22,
		"GI": 23,
		"GL": 18,
		"GR": 27,
		"GT": 28,
		"HN": 28,
		"HR": 21,
		"HU": 28,
		"IE": 22,
		"IL": 23,
		"IQ": 23,
		"IS": 26,
		"IT": 27,
		"JO": 30,
		"KW": 30,
		"KZ": 20,
		"LB": 28,
		"LC": 32,
		"LI": 21,
		"LT": 20,
		"LU": 20,
		"LV": 21,
		"LY": 25,
		"MC": 27,
		"MD": 24,
		"ME": 22,
		"MK": 19,
		"MN": 20,
		"MR": 27,
		"MT": 31,
		"MU": 30,
		"NI": 32,
		"NL": 18,
		"NO": 15,
		"OM": 23,
		"PK": 24,
		"PL": 28,
		"PS": 29,
		"PT": 25,
		"QA": 29,
		"RO": 24,
		"RS": 22,
		"RU": 33,
		"SA": 24,
		"SC": 31,
		"SD": 18,
		"SE": 24,
		"SI": 19,
		"SK": 24,
		"SM": 27,
		"SO": 23,
		"ST": 25,
		"SV": 28,
		"TL": 23,
		"TN": 24,
		"TR": 26,
		"UA": 29,
		"VA": 22,
		"VG": 24,
		"XK": 20,
		"YE": 30,
	}
	// The EPC SEPA scope document lists both BIC and IBAN country codes.
	// This allowlist intentionally follows the IBAN code column because the
	// EPC QR validation is based on the beneficiary IBAN prefix. For example,
	// Guernsey, Jersey, and the Isle of Man are SEPA-reachable via the `GB`
	// IBAN prefix, while Gibraltar uses `GI`.
	sepaSchemeIBANCountryCodes = map[string]struct{}{
		"AD": {},
		"AL": {},
		"AT": {},
		"BE": {},
		"BG": {},
		"CH": {},
		"CY": {},
		"CZ": {},
		"DE": {},
		"DK": {},
		"EE": {},
		"ES": {},
		"FI": {},
		"FR": {},
		"GB": {},
		"GI": {},
		"GR": {},
		"HR": {},
		"HU": {},
		"IE": {},
		"IS": {},
		"IT": {},
		"LI": {},
		"LT": {},
		"LU": {},
		"LV": {},
		"MC": {},
		"MD": {},
		"ME": {},
		"MK": {},
		"MT": {},
		"NL": {},
		"NO": {},
		"PL": {},
		"PT": {},
		"RO": {},
		"RS": {},
		"SE": {},
		"SI": {},
		"SK": {},
		"SM": {},
		"VA": {},
	}
)

// Transfer is the content of an EPC069-12 QR code for a SEPA credit transfer
// in EUR. Encode lays the fields out; it does not validate them.
type Transfer struct {
	BIC         string
	Name        string
	IBAN        string
	AmountCents int64
	Purpose     string
	Text        string
	Information string
}

// Encode returns the QR payload, or an error when it exceeds MaxPayloadBytes.
func (t Transfer) Encode() ([]byte, error) {
	fields := []string{
		"BCD",
		"002",
		"1",
		"SCT",
		t.BIC,
		t.Name,
		t.IBAN,
		"EUR" + formatAmount(t.AmountCents),
		t.Purpose,
		"",
		t.Text,
		t.Information,
	}
	for len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	payload := []byte(strings.Join(fields, "\n"))
	if len(payload) > MaxPayloadBytes {
		return nil, fmt.Errorf("EPC QR code payload exceeds %d bytes", MaxPayloadBytes)
	}
	return payload, nil
}

// CheckText reports why value cannot fill a free-text field of at most
// maxChars characters. An empty value is allowed.
func CheckText(value string, maxChars int) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return errors.New("must be valid UTF-8")
	}
	if strings.ContainsAny(value, "\r\n") {
		return errors.New("line breaks are not allowed")
	}
	if utf8.RuneCountInString(value) > maxChars {
		return fmt.Errorf("exceeds %d characters", maxChars)
	}
	return nil
}

// ValidBIC reports whether bic, in compact form, is an 8- or 11-character BIC.
func ValidBIC(bic string) bool {
	return bicPattern.MatchString(bic)
}

// ValidPurpose reports whether purpose is an EPC purpose code of 1-4 letters
// or digits.
func ValidPurpose(purpose string) bool {
	return purposePattern.MatchString(purpose)
}

func CompactIdentifier(value string) string {
	value = strings.ToUpper(value)
	var compact strings.Builder
	compact.Grow(len(value))
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		compact.WriteRune(r)
	}
	return compact.String()
}

func ValidIBAN(value string) bool {
	if len(value) < 15 || len(value) > 34 {
		return false
	}
	countryCode := value[:2]
	expectedLength, ok := ibanCountryLengths[countryCode]
	if !ok || len(value) != expectedLength {
		return false
	}
	if value[2] < '0' || value[2] > '9' || value[3] < '0' || value[3] > '9' {
		return false
	}
	// ISO 13616 check digits are 98 minus a mod-97 remainder, so 00, 01 and
	// 99 never occur in a valid IBAN even when the checksum works out.
	if checkDigits := value[2:4]; checkDigits < "02" || checkDigits > "98" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	rearranged := value[4:] + value[:4]
	remainder := 0
	for _, r := range rearranged {
		switch {
		case r >= '0' && r <= '9':
			remainder = (remainder*10 + int(r-'0')) % 97
		case r >= 'A' && r <= 'Z':
			digits := int(r-'A') + 10
			remainder = (remainder*10 + digits/10) % 97
			remainder = (remainder*10 + digits%10) % 97
		default:
			return false
		}
	}
	return remainder == 1
}

func SEPASchemeIBAN(value string) bool {
	if len(value) < 2 {
		return false
	}
	_, ok := sepaSchemeIBANCountryCodes[value[:2]]
	return ok
}

func formatAmount(cents int64) string {
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
