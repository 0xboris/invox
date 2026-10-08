package validate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
)

// -i is relative to the working directory invox was given, which need not be
// the process's own.
func TestValidateRunResolvesInputAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: work, Output: filepath.Join(work, "inv.yaml")}); err != nil {
		t.Fatal(err)
	}

	ios, _, out, errOut := iostreams.Test()
	opts := &ValidateOptions{
		IO:          ios,
		Service:     f.Service,
		Getwd:       func() (string, error) { return work, nil },
		InvoicePath: "inv.yaml",
	}
	if err := validateRun(opts); err != nil {
		t.Fatalf("validateRun returned error: %v", err)
	}
	if !strings.HasPrefix(errOut.String(), "Validation OK: ") {
		t.Errorf("stderr = %q, want it to start with %q", errOut.String(), "Validation OK: ")
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}
