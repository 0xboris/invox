package numbering

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func FuzzInvoiceNumberRoundTrip(f *testing.F) {
	for _, seed := range []struct {
		pattern, customerID, customerCode, issueDate string
		counter                                      int64
	}{
		{"{customer_id}-{counter:03}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_code}-{counter:03}", "CUST-001", "APP", "2026-03-06", 7},
		{"{customer_id}-{year}-{counter:04}", "CUST-001", "", "2026-12-31", 12345},
		{"{year}{month}{day}/{customer_code}/{counter}", "1001", "", "2026-01-02", 0},
		{"{counter}.{customer_id}", "A1", "", "2026-03-06", 9223372036854775807},
		// #17: patterns that formatted but did not parse back.
		{" {customer_id}-{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_id}-{counter:03}/{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_code}{counter:03}", "A", "", "2026-03-06", 7},
		// #17: invalid UTF-8 made Parse's regexp.MustCompile panic.
		{"\xff{customer_id}-{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_id}-{counter:999999999}", "CUST-001", "", "2026-03-06", 1},
	} {
		f.Add(seed.pattern, seed.customerID, seed.customerCode, seed.issueDate, seed.counter)
	}

	f.Fuzz(func(t *testing.T, pattern, customerID, customerCode, issueDate string, counter int64) {
		// Callers pass customers.<id>.numbering.code trimmed.
		customerCode = strings.TrimSpace(customerCode)

		// Parsing never panics, whatever the pattern and number.
		_, _ = Parse(pattern, customerID, customerID, customerCode, issueDate)

		// invoice.Host.ResolveNumberingSettings trims the configured pattern, then
		// validates it; only patterns that pass are ever formatted.
		pattern = strings.TrimSpace(pattern)
		if err := (Settings{Pattern: pattern, Start: 1}).Validate(); err != nil {
			return
		}
		// Customer IDs and codes come from YAML, which is valid UTF-8, and
		// IDs are trimmed when an invoice is loaded. Counters are never
		// negative.
		customerID = strings.TrimSpace(customerID)
		if customerID == "" || !utf8.ValidString(customerID) || !utf8.ValidString(customerCode) || counter < 0 {
			return
		}

		number, err := Format(pattern, customerID, customerCode, issueDate, counter)
		if _, dateErr := time.Parse("2006-01-02", issueDate); dateErr != nil {
			if err == nil {
				t.Fatalf("Format accepted invalid issue date %q", issueDate)
			}
			return
		}
		if err != nil {
			t.Fatalf("Format(%q, %q, %q, %d) returned error: %v", pattern, customerID, issueDate, counter, err)
		}

		got, err := Parse(pattern, number, customerID, customerCode, issueDate)
		if err != nil {
			t.Fatalf("pattern %q formatted counter %d as %q, which does not parse back: %v", pattern, counter, number, err)
		}
		if got != counter {
			t.Fatalf("pattern %q formatted counter %d as %q, which parses back as %d", pattern, counter, number, got)
		}
	})
}

func TestParseInvoiceCounterReturnsErrorForInvalidUTF8Pattern(t *testing.T) {
	// regexp.MustCompile panicked on this pattern.
	_, err := Parse("\xff{customer_id}-{counter}", "\xffCUST-001-1", "CUST-001", "", "2026-03-06")
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("Parse error = %v, want an invalid UTF-8 error", err)
	}
}

func TestValidateNumberingSettingsRejectsPatternsThatDoNotRoundTrip(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr string
	}{
		{pattern: " {customer_id}-{counter}", wantErr: "must not start or end with whitespace"},
		{pattern: "{customer_id}-{counter}\t", wantErr: "must not start or end with whitespace"},
		{pattern: "{customer_id}-{counter:03}/{counter}", wantErr: "must contain {counter} only once"},
		{pattern: "{customer_code}{counter:03}", wantErr: `numbering.pattern "{customer_code}{counter:03}" needs a separator between {customer_code} and {counter}, such as "{customer_code}-{counter:03}"`},
		{pattern: "RE-{customer_code}{counter:04}", wantErr: "set numbering.start (or customers.<id>.numbering.start) to continue the sequence"},
		{pattern: "{counter}{customer_id}", wantErr: "needs a separator between {customer_id} and {counter}"},
		{pattern: "{customer_id}-{counter:21}", wantErr: "uses an invalid width; use at most 20"},
		{pattern: "\xff{customer_id}-{counter}", wantErr: "must be valid UTF-8"},
		{pattern: DefaultPattern},
		{pattern: "{customer_id}-{year}-{counter:04}"},
		{pattern: "{customer_code}{year}{counter:20}"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := (Settings{Pattern: tt.pattern, Start: 1}).Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseInvoiceCounterTrimsPatternLikeFormat(t *testing.T) {
	pattern := " {customer_id}-{counter:03} "
	number, err := Format(pattern, "CUST-001", "", "2026-03-06", 7)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	counter, err := Parse(pattern, number, "CUST-001", "", "2026-03-06")
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", number, err)
	}
	if counter != 7 {
		t.Fatalf("Parse(%q) = %d, want 7", number, counter)
	}
}
