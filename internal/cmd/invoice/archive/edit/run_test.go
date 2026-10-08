package edit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// archive edit copies into the working directory invox was given, which
// need not be the process's own.
func TestEditRunCopiesIntoGetwd(t *testing.T) {
	work := t.TempDir()
	draft := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), XDGDataHome: t.TempDir(), ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	invoicePath := filepath.Join(draft, "inv.yaml")
	if _, err := host.CreateNewInvoice(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), draft, host.GlobalInvoiceDefaultsPath(), invoicePath, host.GlobalCustomersPath(), host.GlobalIssuerPath(), "CUST-001", invoice.NewInvoiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := invoice.MarkInvoiceBuilt(invoicePath); err != nil {
		t.Fatal(err)
	}
	result, err := host.ArchiveInvoice(time.Now(), invoicePath, invoice.ArchiveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &EditOptions{
		IO:       ios,
		Host:     func() invoice.Host { return host },
		Getwd:    func() (string, error) { return work, nil },
		Filename: filepath.Base(result.Path),
	}
	if err := editRun(opts); err != nil {
		t.Fatalf("editRun returned error: %v", err)
	}
	copied := filepath.Join(work, filepath.Base(result.Path))
	if _, err := os.Stat(copied); err != nil {
		t.Errorf("archived invoice not copied into the working directory: %v", err)
	}
	if got := out.String(); got != filepath.Base(result.Path)+"\n" {
		t.Errorf("stdout = %q, want %q", got, filepath.Base(result.Path)+"\n")
	}
}
