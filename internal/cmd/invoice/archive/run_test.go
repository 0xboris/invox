package archive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// The invoice is relative to the working directory invox was given, which
// need not be the process's own.
func TestArchiveRunResolvesInputAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), XDGDataHome: t.TempDir(), ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := host.CreateNewInvoice(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), work, host.GlobalInvoiceDefaultsPath(), filepath.Join(work, "inv.yaml"), host.GlobalCustomersPath(), host.GlobalIssuerPath(), "CUST-001", false); err != nil {
		t.Fatal(err)
	}
	if err := invoice.MarkInvoiceBuilt(filepath.Join(work, "inv.yaml")); err != nil {
		t.Fatal(err)
	}

	ios, _, _, _ := iostreams.Test()
	opts := &ArchiveOptions{
		IO:          ios,
		Host:        func() invoice.Host { return host },
		Getwd:       func() (string, error) { return work, nil },
		Now:         time.Now,
		InvoicePath: "inv.yaml",
	}
	if err := archiveRun(context.Background(), opts); err != nil {
		t.Fatalf("archiveRun returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "inv.yaml")); !os.IsNotExist(err) {
		t.Errorf("inv.yaml is still in the working directory (Stat error %v), want it moved to the archive", err)
	}
	archived, err := host.ListArchivedInvoices()
	if err != nil || len(archived) != 1 {
		t.Fatalf("ListArchivedInvoices() = %v, %v; want one archived invoice", archived, err)
	}
}
