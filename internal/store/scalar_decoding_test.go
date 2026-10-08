package store

import (
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

// decodeScalarForTest decodes the value of `value:` in source into field.
func decodeScalarForTest(t *testing.T, source string, field any) error {
	t.Helper()
	document, err := parseYAMLDocumentSource([]byte(source), "test.yaml")
	if err != nil {
		t.Fatalf("parseYAMLDocumentSource returned error: %v", err)
	}
	ok, err := decodeScalar(field, findMappingValue(document.Content[0], "value"))
	if !ok {
		t.Fatalf("decodeScalar does not decode %T", field)
	}
	return err
}

func TestTextKeepsTheWrittenText(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: "value: 01067\n", want: "01067"},
		{source: "value: 0042\n", want: "0042"},
		{source: "value: 12.50\n", want: "12.50"},
		{source: "value: 1e3\n", want: "1e3"},
		{source: "value: 0x1F\n", want: "0x1F"},
		{source: "value: True\n", want: "true"},
		{source: "value: yes\n", want: "yes"},
		{source: "value: ~\n", want: ""},
		{source: "value:\n", want: ""},
		{source: "value: 2026-3-6\n", want: "2026-03-06"},
		{source: "value: 2026-03-06T23:30:00+02:00\n", want: "2026-03-06"},
		{source: "value: \"  padded  \"\n", want: "  padded  "},
		{source: "value: !!binary aGk=\n", want: "hi"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			var got invoice.Text
			if err := decodeScalarForTest(t, tt.source, &got); err != nil {
				t.Fatalf("decodeScalar returned error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Text = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecimalDecoding(t *testing.T) {
	tests := []struct {
		source  string
		want    string
		wantErr string
	}{
		{source: "value: 12\n", want: "12"},
		{source: "value: 12.50\n", want: "25/2"},
		{source: "value: \"0100\"\n", want: "100"},
		{source: "value: 007.50\n", want: "15/2"},
		{source: "value: -3.5\n", want: "-7/2"},
		{source: "value: -0\n", want: "0"},
		{source: "value: \"\"\n", want: "unset"},
		{source: "value: ~\n", want: "unset"},
		{source: "value: 1e3\n", wantErr: "expected a decimal number such as 12 or 12.50, got `1e3`"},
		{source: "value: 1,5\n", wantErr: "expected a decimal number such as 12 or 12.50, got `1,5`"},
		{source: "value: 0x10\n", wantErr: "expected a decimal number such as 12 or 12.50, got `0x10`"},
		{source: "value: {a: 1}\n", wantErr: "expected a decimal number such as 12 or 12.50, got a mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			var got invoice.Decimal
			err := decodeScalarForTest(t, tt.source, &got)
			checkDecodeResult(t, err, tt.wantErr, func() string {
				if !got.IsSet() {
					return "unset"
				}
				return got.Rat().RatString()
			}, tt.want)
		})
	}
}

func TestRateDecoding(t *testing.T) {
	tests := []struct {
		source  string
		want    string
		wantErr string
	}{
		{source: "value: 20\n", want: "20"},
		{source: "value: 7.7\n", want: "77/10"},
		{source: "value: 19%\n", want: "19"},
		{source: "value: \" 20 % \"\n", want: "20"},
		{source: "value: 0\n", want: "0"},
		{source: "value: \"\"\n", want: "unset"},
		{source: "value: twenty\n", wantErr: "expected a number or percent string, got `twenty`"},
		{source: "value: [20]\n", wantErr: "expected a number or percent string, got a list"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			var got invoice.Rate
			err := decodeScalarForTest(t, tt.source, &got)
			checkDecodeResult(t, err, tt.wantErr, func() string {
				if !got.IsSet() {
					return "unset"
				}
				return got.Percent().RatString()
			}, tt.want)
		})
	}
}

func TestDateDecoding(t *testing.T) {
	tests := []struct {
		source  string
		want    string
		wantErr string
	}{
		{source: "value: 2026-03-06\n", want: "2026-03-06"},
		{source: "value: \"2026-03-06\"\n", want: "2026-03-06"},
		{source: "value: 2026-3-6\n", want: "2026-03-06"},
		{source: "value: 2026-03-06T23:30:00+02:00\n", want: "2026-03-06"},
		{source: "value: \"\"\n", want: ""},
		{source: "value: \"2026-3-6\"\n", wantErr: "expected YYYY-MM-DD, got `2026-3-6`"},
		{source: "value: 06.03.2026\n", wantErr: "expected YYYY-MM-DD, got `06.03.2026`"},
		{source: "value: 2026-02-30\n", wantErr: "expected YYYY-MM-DD, got `2026-02-30`"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			var got invoice.Date
			err := decodeScalarForTest(t, tt.source, &got)
			checkDecodeResult(t, err, tt.wantErr, got.String, tt.want)
		})
	}
}

func TestCountDecoding(t *testing.T) {
	tests := []struct {
		source  string
		want    int64
		unset   bool
		wantErr string
	}{
		{source: "value: 30\n", want: 30},
		{source: "value: \"30\"\n", want: 30},
		{source: "value: 030\n", want: 30},
		{source: "value: -1\n", want: -1},
		{source: "value: \"\"\n", unset: true},
		{source: "value: 30.0\n", wantErr: "expected an integer, got `30.0`"},
		{source: "value: 0x1E\n", wantErr: "expected an integer, got `0x1E`"},
		{source: "value: {days: 30}\n", wantErr: "expected an integer, got a mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			var got invoice.Count
			err := decodeScalarForTest(t, tt.source, &got)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeScalar returned error: %v", err)
			}
			if got.IsSet() == tt.unset || got.Int() != tt.want {
				t.Fatalf("Count = %d (set %v), want %d (set %v)", got.Int(), got.IsSet(), tt.want, !tt.unset)
			}
		})
	}
}

func checkDecodeResult(t *testing.T, err error, wantErr string, got func() string, want string) {
	t.Helper()
	if wantErr != "" {
		if err == nil || err.Error() != wantErr {
			t.Fatalf("error = %v, want %q", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("decodeScalar returned error: %v", err)
	}
	if got() != want {
		t.Fatalf("value = %q, want %q", got(), want)
	}
}
