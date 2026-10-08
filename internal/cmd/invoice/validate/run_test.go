package validate

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// -i is relative to the working directory invox was given, which need not be
// the process's own.
func TestValidateRunResolvesInputAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := host.CreateNewInvoice(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), work, host.GlobalInvoiceDefaultsPath(), filepath.Join(work, "inv.yaml"), host.GlobalCustomersPath(), host.GlobalIssuerPath(), "CUST-001", invoice.NewInvoiceOptions{}); err != nil {
		t.Fatal(err)
	}

	ios, _, out, errOut := iostreams.Test()
	opts := &ValidateOptions{
		IO:          ios,
		Host:        func() invoice.Host { return host },
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
