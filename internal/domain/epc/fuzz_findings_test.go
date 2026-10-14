package epc

import (
	"testing"
)

func TestIsValidIBANRejectsOutOfRangeCheckDigits(t *testing.T) {
	// Each of these passes the mod-97 check, but ISO 13616 check digits run
	// from 02 to 98.
	for _, iban := range []string{
		"DE00370400440000000060",
		"DE01370400440000000042",
		"DE99370400440000000024",
	} {
		if ValidIBAN(iban) {
			t.Errorf("ValidIBAN(%q) = true, want false", iban)
		}
	}
	if !ValidIBAN("DE89370400440532013000") {
		t.Error(`ValidIBAN("DE89370400440532013000") = false, want true`)
	}
}
