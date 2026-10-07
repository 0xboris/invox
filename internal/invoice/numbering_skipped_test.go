package invoice

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNextInvoiceNumberReportsSkippedArchiveFiles(t *testing.T) {
	t.Parallel()

	const counterPattern = "{customer_id}-{counter:03}"
	const yearPattern = "{customer_id}-{year}-{counter:03}"

	for _, tc := range []struct {
		name        string
		pattern     string
		files       map[string]string
		wantNumber  string
		wantSkipped []string
	}{
		{
			name:    "same customer non-matching invoice is skipped",
			pattern: counterPattern,
			files: map[string]string{
				"a.yaml":          "customer_id: CUST-001\ninvoice:\n  number: CUST-001-004\n  issue_date: \"2026-03-06\"\n",
				"old-format.yaml": "customer_id: CUST-001\ninvoice:\n  number: CUST-001/0009\n  issue_date: \"2026-03-06\"\n",
			},
			wantNumber:  "CUST-001-005",
			wantSkipped: []string{"old-format.yaml"},
		},
		{
			name:    "other customer non-matching invoice is not skipped",
			pattern: counterPattern,
			files: map[string]string{
				"other.yaml": "customer_id: CUST-002\ninvoice:\n  number: OTHER/17\n  issue_date: \"2026-03-06\"\n",
			},
			wantNumber: "CUST-001-001",
		},
		{
			name:    "files that are not numbered invoices are not skipped",
			pattern: counterPattern,
			files: map[string]string{
				"notes.yaml":     "customer_id: CUST-001\ntodo: call the accountant\n",
				"no-number.yaml": "customer_id: CUST-001\ninvoice:\n  issue_date: \"2026-03-06\"\n",
				"readme.md":      "# Archive\n",
				"a.pdf":          "%PDF-1.4\n",
			},
			wantNumber: "CUST-001-001",
		},
		{
			name:    "invoice from an earlier year is not skipped",
			pattern: yearPattern,
			files: map[string]string{
				"2025.yaml": "customer_id: CUST-001\ninvoice:\n  number: CUST-001-2025-007\n  issue_date: \"2025-05-01\"\n",
			},
			wantNumber: "CUST-001-2026-001",
		},
		{
			name:    "invoice from an earlier year in an old format is skipped",
			pattern: yearPattern,
			files: map[string]string{
				"2025.yaml": "customer_id: CUST-001\ninvoice:\n  number: CUST-001/7\n  issue_date: \"2025-05-01\"\n",
			},
			wantNumber:  "CUST-001-2026-001",
			wantSkipped: []string{"2025.yaml"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			archiveDir := t.TempDir()
			h := writeConfigFile(t, "numbering:\n  pattern: '"+tc.pattern+"'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			for name, source := range tc.files {
				if err := os.WriteFile(filepath.Join(archiveDir, name), []byte(source), 0o644); err != nil {
					t.Fatalf("WriteFile(%s) returned error: %v", name, err)
				}
			}
			var wantSkipped []string
			for _, name := range tc.wantSkipped {
				wantSkipped = append(wantSkipped, filepath.Join(archiveDir, name))
			}

			number, skipped, err := h.NextInvoiceNumber("CUST-001", "2026-03-06", map[string]any{}, 0)
			if err != nil {
				t.Fatalf("NextInvoiceNumber returned error: %v", err)
			}
			if number != tc.wantNumber {
				t.Fatalf("number = %q, want %q", number, tc.wantNumber)
			}
			if !reflect.DeepEqual(skipped, wantSkipped) {
				t.Fatalf("skipped = %q, want %q", skipped, wantSkipped)
			}
		})
	}
}
