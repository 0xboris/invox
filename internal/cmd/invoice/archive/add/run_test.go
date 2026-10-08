package add

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
func TestAddRunResolvesInputAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	dataHome := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), XDGDataHome: dataHome, ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := host.CreateNewInvoice(invoice.NewInvoiceParams{Now: time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), WorkDir: work, DefaultsPath: host.GlobalInvoiceDefaultsPath(), OutputPath: filepath.Join(work, "inv.yaml"), CustomersPath: host.GlobalCustomersPath(), IssuerPath: host.GlobalIssuerPath(), CustomerID: "CUST-001"}); err != nil {
		t.Fatal(err)
	}
	if err := invoice.MarkInvoiceBuilt(filepath.Join(work, "inv.yaml")); err != nil {
		t.Fatal(err)
	}

	ios, _, out, errOut := iostreams.Test()
	opts := &AddOptions{
		IO:          ios,
		Host:        func() invoice.Host { return host },
		Getwd:       func() (string, error) { return work, nil },
		Now:         time.Now,
		InvoicePath: "inv.yaml",
	}
	if err := addRun(context.Background(), opts); err != nil {
		t.Fatalf("addRun returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "inv.yaml")); !os.IsNotExist(err) {
		t.Errorf("inv.yaml is still in the working directory (Stat error %v), want it moved to the archive", err)
	}
	archived, err := host.ListArchivedInvoices()
	if err != nil || len(archived) != 1 {
		t.Fatalf("ListArchivedInvoices() = %v, %v; want one archived invoice", archived, err)
	}
	archivedPath := filepath.Join(dataHome, "invox", "invoices", archived[0].Filename)
	if got := out.String(); got != archivedPath+"\n" {
		t.Errorf("stdout = %q, want %q", got, archivedPath+"\n")
	}
	if got, want := errOut.String(), "Archived inv.yaml -> "+archivedPath+"\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}
