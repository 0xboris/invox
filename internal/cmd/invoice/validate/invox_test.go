package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestValidateAcceptsShortCustomerAndIssuerFlags(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	exitCode, stdout, stderr := x.Run([]string{
		"validate",
		"-i", fx.Invoice,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestValidateSuggestsGlobalDefaultsWhenSupportFilesMissing(t *testing.T) {
	x := clitest.New(t)

	workDir := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "config-home")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"validate",
		"-i", filepath.Join(workDir, "invoice.yaml"),
	})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	expected := filepath.Join(configHome, "invox", "customers.yaml")
	if !strings.Contains(stderr, expected) {
		t.Fatalf("stderr %q does not mention global customers path %q", stderr, expected)
	}
}

func TestValidateReportsUnknownCustomerWithHint(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	x.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "invoice.yaml"), []byte(strings.Replace(string(source), "customer_id: CUST-001", "customer_id: NOPE-1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := x.Run([]string{"validate", "-i", "invoice.yaml", "-c", fx.Customers, "-u", fx.Issuer})

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	want := "error: invoice.yaml: unknown customer_id `NOPE-1`\nRun 'invox customer list' to see the customer IDs.\n"
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestValidateReportsCustomerProblemsWithoutFieldNoise(t *testing.T) {
	tests := []struct {
		name    string
		replace string
		with    string
		extra   string
		want    string
	}{
		{
			name:    "missing customer_id",
			replace: "customer_id: CUST-001\n",
			with:    "",
			want:    "error: invoice.yaml: missing `customer_id`\n",
		},
		{
			name:    "customer is not a mapping",
			replace: "customer_id: CUST-001",
			with:    "customer_id: CUST-SCALAR",
			extra:   "\nCUST-SCALAR: just a string\n",
			want:    "error: customers.yaml:15: customer `CUST-SCALAR` must be a mapping\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := clitest.New(t)

			fx := testfixture.WriteContext(t)
			source, err := os.ReadFile(fx.Invoice)
			if err != nil {
				t.Fatal(err)
			}
			customers, err := os.ReadFile(fx.Customers)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			x.Chdir(dir)
			if !strings.Contains(string(source), tt.replace) {
				t.Fatalf("fixture invoice has no %q", tt.replace)
			}
			if err := os.WriteFile(filepath.Join(dir, "invoice.yaml"), []byte(strings.Replace(string(source), tt.replace, tt.with, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "customers.yaml"), append(customers, tt.extra...), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := x.Run([]string{"validate", "-i", "invoice.yaml", "-c", "customers.yaml", "-u", fx.Issuer})

			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
		})
	}
}

func TestValidateWarnsWhenNumberIsAlreadyArchived(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	exitCode, stdout, stderr := x.Run([]string{"validate", "-i", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := "warning: invoice number CUST-001-001 is already used by archived invoice " + archivedPath +
		"; run 'invox increment -i " + fx.Invoice + "' before archiving\n" +
		"Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestValidateDoesNotWarnForUniqueNumber(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteNumberedInvoice(t, archiveDir, "other.yaml", "CUST-001-002", "archived")

	exitCode, stdout, stderr := x.Run([]string{"validate", "-i", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 €\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}
