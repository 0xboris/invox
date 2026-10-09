package increment_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestIncrementRequiresInput(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"increment"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required input: INVOICE.yaml or -i, --input") {
		t.Fatalf("stderr %q does not contain missing input message", stderr)
	}
}

func TestIncrementUpdatesInvoiceNumber(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	invoicePath := filepath.Join(workDir, "invoice.yaml")
	archiveDir := t.TempDir()
	if err := os.WriteFile(invoicePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-009
  issue_date: 2026-03-06
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteArchivedInvoice(t, archiveDir, "2026-03-05.yaml", "CUST-001-011")

	exitCode, stdout, stderr := x.Run([]string{
		"increment",
		"-i", invoicePath,
		"-c", draft.Customers,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Incremented " + invoicePath + " for CUST-001: CUST-001-009 -> CUST-001-012\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != invoicePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, invoicePath+"\n")
	}

	updated, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(updated), "number: CUST-001-012") {
		t.Fatalf("invoice file does not contain incremented number: %q", string(updated))
	}
}

// increment reads the whole invoice but needs only customer_id,
// invoice.number and invoice.issue_date to decode: a bad value elsewhere
// does not stop it, a bad issue date does, with its line.
func TestIncrementNeedsOnlyTheFieldsNumberingReads(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.Chdir(workDir)
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(filepath.Join(workDir, "archive")) + "\n")

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
			testfixture.WriteFile(t, filepath.Join(workDir, "invoice.yaml"), "customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: "+tt.issueDate+"\n  paid_amount: "+tt.paidAmount+"\n")
			exitCode, stdout, stderr := x.Run([]string{"increment", "invoice.yaml", "-c", draft.Customers})
			if exitCode != tt.exitCode || stdout != tt.stdout || stderr != tt.stderr {
				t.Fatalf("increment = exit %d, stdout %q, stderr %q; want exit %d, stdout %q, stderr %q", exitCode, stdout, stderr, tt.exitCode, tt.stdout, tt.stderr)
			}
		})
	}
}

func TestIncrementWritesThroughSymlinkedInvoice(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	targetPath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(targetPath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: 2026-03-06\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(t.TempDir(), "linked.yaml")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(t.TempDir()) + "\n")

	exitCode, stdout, stderr := x.Run([]string{"increment", "-i", linkPath, "-c", draft.Customers})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Incremented " + linkPath + " for CUST-001: CUST-001-009 -> CUST-001-010\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != linkPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, linkPath+"\n")
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s was replaced by a regular file", linkPath)
	}
	if got, err := os.Readlink(linkPath); err != nil || got != targetPath {
		t.Fatalf("Readlink(%s) = %q, %v, want %q", linkPath, got, err, targetPath)
	}
	updated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "number: CUST-001-010") {
		t.Fatalf("link target was not updated:\n%s", updated)
	}
}
