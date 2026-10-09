//go:build !windows

package increment_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestIncrementKeepsPrivateInvoiceMode(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: 2026-03-06\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(t.TempDir()) + "\n")

	exitCode, stdout, stderr := x.Run([]string{"increment", "-i", invoicePath, "-c", draft.Customers})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Incremented " + invoicePath + " for CUST-001: CUST-001-009 -> CUST-001-010\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != invoicePath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, invoicePath+"\n")
	}
	testfixture.AssertFileMode(t, invoicePath, 0o600)
}
