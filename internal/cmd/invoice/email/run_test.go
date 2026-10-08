package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// The invoice, its PDF and -o are relative to the working directory invox
// was given, which need not be the process's own.
func TestEmailRunResolvesPathsAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := host.CreateNewInvoice(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), work, host.GlobalInvoiceDefaultsPath(), filepath.Join(work, "inv.yaml"), host.GlobalCustomersPath(), host.GlobalIssuerPath(), "CUST-001", false); err != nil {
		t.Fatal(err)
	}
	if err := invoice.MarkInvoiceBuilt(filepath.Join(work, "inv.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "inv.pdf"), []byte("%PDF\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	stub := run.NewStub(t)
	var opened string
	stub.Register("xdg-open", func(cmd run.Cmd) error {
		opened = cmd.Args[0]
		return nil
	})
	opts := &EmailOptions{
		IO:          ios,
		Opener:      opener.New(stub, ios, "linux"),
		Host:        func() invoice.Host { return host },
		Getwd:       func() (string, error) { return work, nil },
		Now:         time.Now,
		InvoicePath: "inv.yaml",
		OutputPath:  "draft.eml",
	}
	if err := emailRun(context.Background(), opts, true); err != nil {
		t.Fatalf("emailRun returned error: %v", err)
	}
	want := filepath.Join(work, "draft.eml")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("draft.eml not written to the working directory: %v", err)
	}
	if opened != want {
		t.Errorf("opened %q, want %q", opened, want)
	}
	if got := out.String(); got != "draft.eml\n" {
		t.Errorf("stdout = %q, want %q", got, "draft.eml\n")
	}
}
