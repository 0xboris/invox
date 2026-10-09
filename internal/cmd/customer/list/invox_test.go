package list_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestCustomerListPrintsCustomers(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	exitCode, stdout, stderr := x.Run([]string{
		"customer",
		"list",
		"-c", fx.Customers,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "CUST-001\tAppsters GmbH\tactive") {
		t.Fatalf("stdout %q does not contain expected customer row", stdout)
	}
}

func TestCustomerListPreservesUnquotedNumericLookingCustomerID(t *testing.T) {
	x := clitest.New(t)

	customersPath := filepath.Join(t.TempDir(), "customers.yaml")
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
0021:
  status: active
  name: Appsters GmbH
  email: office@appsters.at
  address:
    street: Griesgasse 19
    postal_code: "9020"
    city: Klagenfurt
    country: Oesterreich
  tax:
    vat_tax_id: ATU80037005
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"customer",
		"list",
		"-c", customersPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "0021\tAppsters GmbH\tactive") {
		t.Fatalf("stdout %q does not contain preserved customer ID", stdout)
	}
}
