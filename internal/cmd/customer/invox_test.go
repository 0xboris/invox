package customer_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestCustomerHelpShowsCustomerSubcommands(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"customer", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox customer <subcommand> [flags]",
		"  edit  Open customers.yaml in your editor\n",
		"  list  List all customers from customers.yaml\n",
		"Customer fields:",
		"<customer>.tax.default_vat_rate",
		"<customer>.billing.send_invoice_to",
		"<customer>.legal_company_name",
		"customers.yaml example:",
		"CUST-001:",
		"send_invoice_to: accounting@appsters.example",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}
