package epc

import (
	"strings"
	"testing"
)

func TestTransferEncode(t *testing.T) {
	got, err := Transfer{
		BIC:         "BKAUATWW",
		Name:        "Boris Consulting",
		IBAN:        "AT611904300234573201",
		AmountCents: 12050,
		Text:        "INV-1",
	}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	want := "BCD\n002\n1\nSCT\nBKAUATWW\nBoris Consulting\nAT611904300234573201\nEUR120.50\n\n\nINV-1"
	if string(got) != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}

	got, err = Transfer{Name: "A", IBAN: "AT611904300234573201", AmountCents: 1}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if want := "BCD\n002\n1\nSCT\n\nA\nAT611904300234573201\nEUR0.01"; string(got) != want {
		t.Fatalf("Encode() with empty trailing fields = %q, want %q", got, want)
	}

	_, err = Transfer{Name: strings.Repeat("x", 200), IBAN: "AT611904300234573201", AmountCents: 1, Text: strings.Repeat("y", 140)}.Encode()
	if err == nil || err.Error() != "EPC QR code payload exceeds 331 bytes" {
		t.Fatalf("Encode() of an oversized payload: err = %v", err)
	}
}

func TestCheckText(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"", ""},
		{"Rechnung 1", ""},
		{"\xff", "must be valid UTF-8"},
		{"a\nb", "line breaks are not allowed"},
		{strings.Repeat("ä", 10), ""},
		{strings.Repeat("ä", 11), "exceeds 10 characters"},
	}
	for _, tt := range tests {
		got := ""
		if err := CheckText(tt.value, 10); err != nil {
			got = err.Error()
		}
		if got != tt.want {
			t.Errorf("CheckText(%q, 10) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestValidBICAndPurpose(t *testing.T) {
	for bic, want := range map[string]bool{"BKAUATWW": true, "BKAUATWWXXX": true, "BKAUAT": false, "bkauatww": false} {
		if got := ValidBIC(bic); got != want {
			t.Errorf("ValidBIC(%q) = %v, want %v", bic, got, want)
		}
	}
	for purpose, want := range map[string]bool{"SUPP": true, "GDDS1": false, "": false, "A-1": false} {
		if got := ValidPurpose(purpose); got != want {
			t.Errorf("ValidPurpose(%q) = %v, want %v", purpose, got, want)
		}
	}
}
