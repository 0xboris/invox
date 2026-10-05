package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestEmailReportsPDFWithoutInvoiceAsRuntimeError(t *testing.T) {
	isolateUserDirs(t)
	customersPath, issuerPath, _ := writeBuiltEmailFixture(t)
	stubEmailOpeners(t)
	dir := t.TempDir()
	chdirForTest(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "orphan.pdf"), []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"email", "orphan.pdf", "-c", customersPath, "-u", issuerPath})

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
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStderr string
	}{
		{"success", nil, 0, ""},
		{"help shown", flag.ErrHelp, 0, ""},
		{"runtime error", errors.New("disk full"), 1, "error: disk full\n"},
		{"usage error", cmdutil.FlagErrorf("customer list", "unexpected arguments: x"), 2, "error: unexpected arguments: x\nRun 'invox customer list --help' for usage.\n"},
		{"root usage error", cmdutil.FlagErrorf("", "missing subcommand"), 2, "error: missing subcommand\nRun 'invox --help' for usage.\n"},
		{"wrapped usage error", fmt.Errorf("parse: %w", cmdutil.FlagErrorf("new", "bad")), 2, "error: bad\nRun 'invox new --help' for usage.\n"},
		{"already reported", cmdutil.SilentError, 1, ""},
		{"cancelled", cmdutil.CancelError, 2, ""},
		{"external program", &cmdutil.ExecError{Program: "tectonic", Code: 3, Err: errors.New("exit status 3")}, 3, ""},
		{"external program killed", &cmdutil.ExecError{Program: "tectonic", Code: -1, Err: errors.New("signal: killed")}, 1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, stderr := iostreams.Test()
			if got := exitCode(ios, tt.err); got != tt.wantCode {
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

	exitCode(ios, fmt.Errorf("%s and %s", filepath.Join(cwd, "sub", "a.yaml"), outside))

	if want := "error: " + filepath.Join("sub", "a.yaml") + " and " + outside + "\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestTectonicInstallHintDependsOnOS(t *testing.T) {
	tests := map[string]string{
		"darwin":  "Install it with 'brew install tectonic', then rerun this command.",
		"linux":   "Install it from https://tectonic-typesetting.github.io, then rerun this command.",
		"windows": "Install it from https://tectonic-typesetting.github.io, then rerun this command.",
	}
	old := hostOS
	t.Cleanup(func() { hostOS = old })
	for goos, want := range tests {
		hostOS = goos
		ios, _, _, stderr := iostreams.Test()

		exitCode(ios, fmt.Errorf("build: %w", invoice.ErrTectonicNotFound))

		if got, want := stderr.String(), "error: build: tectonic not found in PATH\n"+want+"\n"; got != want {
			t.Errorf("GOOS=%s: stderr = %q, want %q", goos, got, want)
		}
	}
}
