package money

import (
	"testing"
)

func TestParseDecimalGrammar(t *testing.T) {
	tests := []struct {
		text   string
		want   string
		wantOK bool
	}{
		{text: "12", want: "12", wantOK: true},
		{text: "12.50", want: "25/2", wantOK: true},
		{text: "-3.5", want: "-7/2", wantOK: true},
		{text: " 7 ", want: "7", wantOK: true},
		{text: "010", want: "10", wantOK: true},
		{text: "-0", want: "0", wantOK: true},
		{text: "0x10"},
		{text: "0o10"},
		{text: "0b10"},
		{text: "1/3"},
		{text: "010/1"},
		{text: "1e3"},
		{text: ".5"},
		{text: "5."},
		{text: "+5"},
		{text: "1_000"},
		{text: "1,5"},
		{text: ""},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := ParseDecimal(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ParseDecimal(%q) ok = %v, want %v", tt.text, ok, tt.wantOK)
			}
			if ok && got.RatString() != tt.want {
				t.Fatalf("ParseDecimal(%q) = %s, want %s", tt.text, got.RatString(), tt.want)
			}
		})
	}
}
