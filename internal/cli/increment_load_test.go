package cli

import (
	"path/filepath"
	"testing"
)

// increment reads the whole invoice but needs only customer_id,
// invoice.number and invoice.issue_date to decode: a bad value elsewhere
// does not stop it, a bad issue date does, with its line.
func TestIncrementNeedsOnlyTheFieldsNumberingReads(t *testing.T) {
	customersPath, _, _ := writeDraftFixtures(t)
	workDir := t.TempDir()
	chdirForTest(t, workDir)
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(filepath.Join(workDir, "archive"))+"\n")

	tests := []struct {
		name, issueDate, paidAmount string
		exitCode                    int
		stdout, stderr              string
	}{
		{
			name: "unrelated bad value", issueDate: "2026-03-06", paidAmount: "lots",
			stdout: "invoice.yaml\n",
			stderr: "Incremented invoice.yaml for CUST-001: CUST-001-009 -> CUST-001-010\n",
		},
		{
			name: "bad issue date", issueDate: "2026-13-45", paidAmount: "0",
			exitCode: 1,
			stderr:   "error: invoice.yaml:4: invoice.issue_date: expected YYYY-MM-DD, got `2026-13-45`\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeTestFile(t, filepath.Join(workDir, "invoice.yaml"), "customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: "+tt.issueDate+"\n  paid_amount: "+tt.paidAmount+"\n")
			exitCode, stdout, stderr := captureRun(t, []string{"increment", "invoice.yaml", "-c", customersPath})
			if exitCode != tt.exitCode || stdout != tt.stdout || stderr != tt.stderr {
				t.Fatalf("increment = exit %d, stdout %q, stderr %q; want exit %d, stdout %q, stderr %q", exitCode, stdout, stderr, tt.exitCode, tt.stdout, tt.stderr)
			}
		})
	}
}
