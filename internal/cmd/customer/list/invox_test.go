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

func TestCustomerListOutput(t *testing.T) {
	x := clitest.New(t)

	dir := t.TempDir()
	x.Chdir(dir)
	testfixture.WriteFile(t, filepath.Join(dir, "customers.yaml"), `ACME:
  name: "Acme\tTools\nLtd \e[31mred\e[0m C:\\x"
  status: active
CTRL:
  name: "a\x01b\bc\x7fd\Ne\rf\u202eg\u2066h\u200bi\u00adj"
EMOJI:
  name: "🚀⭐✅👍🏽"
  status: active
LONG:
  name: Very Long Company Name Gesellschaft mit beschraenkter Haftung
  status: active
WIDE:
  name: 株式会社テスト
  status: inactive
`)
	testfixture.WriteFile(t, filepath.Join(dir, "empty.yaml"), "{}\n")

	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name: "pipe",
			Args: []string{"customer", "list", "-c", "customers.yaml"},
			WantStdout: "ACME\tAcme\\tTools\\nLtd red C:\\\\x\tactive\n" +
				"CTRL\tabcde\\rfghij\t\n" +
				"EMOJI\t🚀⭐✅👍🏽\tactive\n" +
				"LONG\tVery Long Company Name Gesellschaft mit beschraenkter Haftung\tactive\n" +
				"WIDE\t株式会社テスト\tinactive\n",
		},
		{
			Name: "terminal",
			TTY:  true,
			Args: []string{"customer", "list", "-c", "customers.yaml"},
			WantStdout: "ID     NAME" + strings.Repeat(" ", 38) + "STATUS\n" +
				"ACME   Acme Tools Ltd red C:\\x" + strings.Repeat(" ", 19) + "active\n" +
				"CTRL   abcde fghij\n" +
				"EMOJI  🚀⭐✅👍🏽" + strings.Repeat(" ", 34) + "active\n" +
				"LONG   Very Long Company Name Gesellschaft mit…  active\n" +
				"WIDE   株式会社テスト" + strings.Repeat(" ", 28) + "inactive\n",
		},
		{
			Name: "pipe empty",
			Args: []string{"customer", "list", "-c", "empty.yaml"},
		},
		{
			Name:       "terminal empty",
			TTY:        true,
			Args:       []string{"customer", "list", "-c", "empty.yaml"},
			WantStderr: "No customers found in empty.yaml\n",
		},
	})
}
