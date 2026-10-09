package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
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

func TestExitCodeMapsErrorTypes(t *testing.T) {
	root := &cobra.Command{Use: "invox"}
	customer := &cobra.Command{Use: "customer"}
	customerList := &cobra.Command{Use: "list"}
	customer.AddCommand(customerList)
	newCmd := &cobra.Command{Use: "new"}
	validate := &cobra.Command{Use: "validate"}
	root.AddCommand(customer, newCmd, validate)

	tests := []struct {
		name       string
		cmd        *cobra.Command
		err        error
		wantCode   int
		wantStderr string
	}{
		{"success", root, nil, 0, ""},
		{"help shown", root, flag.ErrHelp, 0, ""},
		{"help flag with a value", validate, &cmdutil.FlagError{Err: flag.ErrHelp}, 2, "error: flag: help requested\nRun 'invox validate --help' for usage.\n"},
		{"runtime error", newCmd, errors.New("disk full"), 1, "error: disk full\n"},
		{"usage error", customerList, cmdutil.FlagErrorf("unexpected arguments: x"), 2, "error: unexpected arguments: x\nRun 'invox customer list --help' for usage.\n"},
		{"root usage error", root, cmdutil.FlagErrorf("missing subcommand"), 2, "error: missing subcommand\nRun 'invox --help' for usage.\n"},
		{"global flag error", customerList, &cmdutil.FlagError{Err: errors.New("flag needs an argument: --config"), Root: true}, 2, "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n"},
		{"wrapped usage error", newCmd, fmt.Errorf("parse: %w", cmdutil.FlagErrorf("bad")), 2, "error: bad\nRun 'invox new --help' for usage.\n"},
		{"already reported", root, cmdutil.SilentError, 1, ""},
		{"cancelled", root, cmdutil.CancelError, 2, ""},
		{"external program", root, &cmdutil.ExecError{Program: "tectonic", Code: 3, Err: errors.New("exit status 3")}, 1, "error: tectonic exited with status 3\n"},
		{"external program killed", root, &cmdutil.ExecError{Program: "tectonic", Code: -1, Err: errors.New("signal: killed")}, 1, "error: tectonic failed: signal: killed\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, stderr := iostreams.Test()
			if got := exitCode(ios, tt.cmd, tt.err); got != tt.wantCode {
				t.Errorf("exitCode = %d, want %d", got, tt.wantCode)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}

func TestRuntimeErrorShowsPathsRelativeToWorkingDir(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "b.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, nil, fmt.Errorf("%s and %s", filepath.Join(cwd, "sub", "a.yaml"), outside))

	if want := "error: " + filepath.Join("sub", "a.yaml") + " and " + outside + "\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestMissingTectonicPrintsInstallHint(t *testing.T) {
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, nil, fmt.Errorf("build: %w", &billing.ToolMissingError{Tool: "tectonic", Hint: "Install it with 'brew install tectonic', then rerun this command."}))

	if want := "error: build: tectonic not found in PATH\nInstall it with 'brew install tectonic', then rerun this command.\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
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

func TestRuntimeErrorKeepsPathsThatOnlyContainWorkingDir(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	mirror := filepath.Join(t.TempDir(), "mirror") + filepath.Join(cwd, "bad.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, nil, fmt.Errorf("%s: invalid", mirror))

	if want := "error: " + mirror + ": invalid\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}
