package epc

import (
	"testing"
)

func TestCompactEPCAccountIdentifierRemovesUnicodeWhitespace(t *testing.T) {
	got := CompactIdentifier(" \tAT61\u00a01904 3002\t3457 3201\n")
	want := "AT611904300234573201"
	if got != want {
		t.Fatalf("CompactIdentifier returned %q, want %q", got, want)
	}
}

func TestIsValidIBANRejectsUnknownCountryCodeAndWrongLength(t *testing.T) {
	tests := []struct {
		name  string
		iban  string
		valid bool
	}{
		{name: "valid Austria", iban: "AT611904300234573201", valid: true},
		{name: "valid Poland", iban: "PL61109010140000071219812874", valid: true},
		{name: "unknown country code", iban: "ZZ6600000000000", valid: false},
		{name: "wrong Austria length", iban: "AT61190430023457320", valid: false},
		{name: "non-numeric check digits", iban: "ATAA1904300234573201", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidIBAN(tt.iban); got != tt.valid {
				t.Fatalf("ValidIBAN(%q) = %v, want %v", tt.iban, got, tt.valid)
			}
		})
	}
}

func TestIsSEPASchemeIBAN(t *testing.T) {
	tests := []struct {
		name  string
		iban  string
		valid bool
	}{
		{name: "sepa Austria", iban: "AT611904300234573201", valid: true},
		{name: "sepa GB prefix covers Crown Dependencies", iban: "GB29NWBK60161331926819", valid: true},
		{name: "sepa Gibraltar", iban: "GI75NWBK000000007099453", valid: true},
		{name: "sepa Poland", iban: "PL61109010140000071219812874", valid: true},
		{name: "non-sepa Brazil", iban: "BR150000000000000000000000000", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SEPASchemeIBAN(tt.iban); got != tt.valid {
				t.Fatalf("SEPASchemeIBAN(%q) = %v, want %v", tt.iban, got, tt.valid)
			}
		})
	}
}

func TestSEPASchemeIBANCountryCodesMatchCurrentEPCIBANCodeList(t *testing.T) {
	expected := []string{
		"AD", "AL", "AT", "BE", "BG", "CH", "CY", "CZ", "DE", "DK",
		"EE", "ES", "FI", "FR", "GB", "GI", "GR", "HR", "HU", "IE",
		"IS", "IT", "LI", "LT", "LU", "LV", "MC", "MD", "ME", "MK",
		"MT", "NL", "NO", "PL", "PT", "RO", "RS", "SE", "SI", "SK",
		"SM", "VA",
	}

	if got, want := len(sepaSchemeIBANCountryCodes), len(expected); got != want {
		t.Fatalf("len(sepaSchemeIBANCountryCodes) = %d, want %d", got, want)
	}
	for _, code := range expected {
		if _, ok := sepaSchemeIBANCountryCodes[code]; !ok {
			t.Fatalf("sepaSchemeIBANCountryCodes is missing %q", code)
		}
	}
}

func TestSEPASchemeIBANCountryCodesHaveIBANLengthDefinitions(t *testing.T) {
	for code := range sepaSchemeIBANCountryCodes {
		if _, ok := ibanCountryLengths[code]; !ok {
			t.Fatalf("ibanCountryLengths is missing SEPA IBAN country code %q", code)
		}
	}
}
