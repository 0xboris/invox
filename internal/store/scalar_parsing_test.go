package store

import (
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

func TestDecodeYAMLKeepsSourceTextOfNumericScalars(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "postal code with leading zero", source: "value: 01067\n", want: "01067"},
		{name: "invoice number with leading zero", source: "value: 0042\n", want: "0042"},
		{name: "hex-looking id", source: "value: 0x10\n", want: "0x10"},
		{name: "octal-looking id", source: "value: 0o17\n", want: "0o17"},
		{name: "phone number", source: "value: 0043123456\n", want: "0043123456"},
		{name: "float keeps trailing zeros", source: "value: 12.50\n", want: "12.50"},
		{name: "exponent", source: "value: 1e3\n", want: "1e3"},
		{name: "plain integer", source: "value: 1010\n", want: "1010"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeForTest[struct {
				Value invoice.Text `yaml:"value"`
			}](t, tt.source).Value
			if got != invoice.Text(tt.want) {
				t.Fatalf("value = %#v, want %q", got, tt.want)
			}
		})
	}
}
