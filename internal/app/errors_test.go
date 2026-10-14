package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmailReportsPDFWithoutInvoiceAsRuntimeError(t *testing.T) {
	isolateUserDirs(t)
	customersPath, issuerPath, _ := writeBuiltEmailFixture(t)
	f, _ := testFactory(t)
	dir := t.TempDir()
	chdirForTest(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "orphan.pdf"), []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"email", "orphan.pdf", "-c", customersPath, "-u", issuerPath})

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	want := "error: orphan.pdf: no matching invoice YAML found next to the PDF or in archive.dir\n"
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestValidateReportsUnknownCustomerWithHint(t *testing.T) {
	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	chdirForTest(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "invoice.yaml"), []byte(strings.Replace(string(source), "customer_id: CUST-001", "customer_id: NOPE-1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"validate", "-i", "invoice.yaml", "-c", customersPath, "-u", issuerPath})

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
			customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
			source, err := os.ReadFile(invoicePath)
			if err != nil {
				t.Fatal(err)
			}
			customers, err := os.ReadFile(customersPath)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			chdirForTest(t, dir)
			if !strings.Contains(string(source), tt.replace) {
				t.Fatalf("fixture invoice has no %q", tt.replace)
			}
			if err := os.WriteFile(filepath.Join(dir, "invoice.yaml"), []byte(strings.Replace(string(source), tt.replace, tt.with, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "customers.yaml"), append(customers, tt.extra...), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := captureRun(t, []string{"validate", "-i", "invoice.yaml", "-c", "customers.yaml", "-u", issuerPath})

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

func TestBuildExitsOneWhenTectonicExitsTwo(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicExit2)

	exitCode, stdout, stderr := captureRun(t, []string{"build", "-i", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath})

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if want := "fake tectonic: forced failure\nerror: tectonic exited with status 2\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}
