package edit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestCustomerEditOpensCustomersFile(t *testing.T) {
	x := clitest.New(t)

	customersPath := filepath.Join(t.TempDir(), "customers.yaml")
	if err := os.WriteFile(customersPath, []byte("CUST-001: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}
	openedPath := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{"customer", "edit", "-c", customersPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if *openedPath != customersPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, customersPath)
	}
	if want := "Opened " + customersPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestCustomerEditHelpShowsEditUsage(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"customer", "edit", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Open customers.yaml in your editor.",
		"invox customer edit [flags]",
		"-c, --customers string",
		"<customer>.name",
		"<customer>.email",
		"<customer>.billing.send_invoice_to",
		"<customer>.tax.default_vat_rate",
		"<customer>.billing.currency",
		"<customer>.numbering.code",
		"<customer>.numbering.start",
		"$ invox customer edit -c customers.yaml",
		"customers.yaml example:",
		"# legal_company_name: Appsters GmbH",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestCustomerConfigReportsIndentedTopLevelConfig(t *testing.T) {
	x := clitest.New(t)

	x.WriteConfig(" numbering:\n  pattern: '{customer_id}-{counter:03}'\npaths:\n  customers: '~/customers.yaml'\n")

	exitCode, stdout, stderr := x.Run([]string{"customer", "edit"})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "top-level keys must not be indented") {
		t.Fatalf("stderr %q does not contain indentation error", stderr)
	}
}
