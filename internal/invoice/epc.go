package invoice

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/0xboris/invox/internal/money"
)

const (
	epcQRMaxPayloadBytes = 331
	epcQRMaxNameChars    = 70
	epcQRMaxPurposeChars = 4
	epcQRMaxTextChars    = 140
	epcQRMaxInfoChars    = 70
	epcQRMaxAmountCents  = 99999999999
)

var (
	epcPurposePattern  = regexp.MustCompile(`^[A-Za-z0-9]{1,4}$`)
	epcBICPattern      = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
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

func epcQRCodeEligible(ctx *Context) bool {
	return ctx != nil && ctx.OutstandingCents > 0 && strings.TrimSpace(ctx.Currency) == "EUR"
}

func buildEPCPayload(ctx *Context) ([]byte, error) {
	if strings.TrimSpace(ctx.Currency) != "EUR" {
		return nil, fmt.Errorf("EPC QR code requires billing.currency EUR, got `%s`", ctx.Currency)
	}
	// EPC amounts run from 0.01 to 999999999.99.
	if ctx.OutstandingCents <= 0 {
		return nil, errors.New("invoice.outstanding_amount: EPC QR code requires an amount above zero")
	}
	if ctx.OutstandingCents > ctx.TotalCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds total `%s`", money.FormatCents(ctx.OutstandingCents), money.FormatCents(ctx.TotalCents))
	}
	if ctx.OutstandingCents > epcQRMaxAmountCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds EPC QR maximum `%s`", money.FormatCents(ctx.OutstandingCents), "999999999,99")
	}

	name := ctx.Payment.EPCQR.Name.Trim()
	if name == "" {
		name = ctx.Company.LegalCompanyName.Trim()
	}
	if name == "" {
		return nil, errors.New("issuer.payment.epc_qr.name: missing value")
	}

	iban := compactEPCAccountIdentifier(string(ctx.Payment.IBAN))
	if !isValidIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: invalid IBAN `%s`", ctx.Payment.IBAN)
	}
	if !isSEPASchemeIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: IBAN `%s` is outside the current SEPA scheme scope", ctx.Payment.IBAN)
	}

	bic := compactEPCAccountIdentifier(string(ctx.Payment.BIC))
	if bic != "" && !epcBICPattern.MatchString(bic) {
		return nil, fmt.Errorf("issuer.payment.bic: invalid BIC `%s`", ctx.Payment.BIC)
	}

	purpose := strings.ToUpper(ctx.Payment.EPCQR.Purpose.Trim())
	if purpose != "" && !epcPurposePattern.MatchString(purpose) {
		return nil, fmt.Errorf("issuer.payment.epc_qr.purpose: expected 1-4 letters or digits, got `%s`", purpose)
	}

	text := ctx.Payment.EPCQR.Text.Trim()
	if text == "" {
		text = ctx.Invoice.Number.Trim()
	}
	information := ctx.Payment.EPCQR.Information.Trim()

	for _, field := range []struct {
		label    string
		value    string
		maxChars int
	}{
		{label: "issuer.payment.epc_qr.name", value: name, maxChars: epcQRMaxNameChars},
		{label: "issuer.payment.epc_qr.text", value: text, maxChars: epcQRMaxTextChars},
		{label: "issuer.payment.epc_qr.information", value: information, maxChars: epcQRMaxInfoChars},
	} {
		if err := validateEPCTextField(field.label, field.value, field.maxChars); err != nil {
			return nil, err
		}
	}

	amount := "EUR" + formatEPCAmount(ctx.OutstandingCents)
	fields := []string{
		"BCD",
		"002",
		"1",
		"SCT",
		bic,
		name,
		iban,
		amount,
		purpose,
		"",
		text,
		information,
	}
	for len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	payload := strings.Join(fields, "\n")
	payloadBytes := []byte(payload)
	if len(payloadBytes) > epcQRMaxPayloadBytes {
		return nil, fmt.Errorf("EPC QR code payload exceeds %d bytes", epcQRMaxPayloadBytes)
	}
	return payloadBytes, nil
}

func validateEPCTextField(label, value string, maxChars int) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s: must be valid UTF-8", label)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s: line breaks are not allowed", label)
	}
	if utf8.RuneCountInString(value) > maxChars {
		return fmt.Errorf("%s: exceeds %d characters", label, maxChars)
	}
	return nil
}

func compactEPCAccountIdentifier(value string) string {
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

func isValidIBAN(value string) bool {
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

func isSEPASchemeIBAN(value string) bool {
	if len(value) < 2 {
		return false
	}
	_, ok := sepaSchemeIBANCountryCodes[value[:2]]
	return ok
}

func formatEPCAmount(cents int64) string {
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
