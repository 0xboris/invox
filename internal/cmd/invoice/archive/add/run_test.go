package add

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// The invoice is relative to the working directory invox was given, which
// need not be the process's own.
func TestAddRunResolvesInputAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	dataHome := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"XDG_DATA_HOME": dataHome, "INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: work, Output: filepath.Join(work, "inv.yaml")}); err != nil {
		t.Fatal(err)
	}
	if err := factorytest.SetStatus(filepath.Join(work, "inv.yaml"), invoice.Built); err != nil {
		t.Fatal(err)
	}

	ios, _, out, errOut := iostreams.Test()
	opts := &AddOptions{
		IO:          ios,
		Service:     f.Service,
		Getwd:       func() (string, error) { return work, nil },
		InvoicePath: "inv.yaml",
	}
	if err := addRun(context.Background(), opts); err != nil {
		t.Fatalf("addRun returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "inv.yaml")); !os.IsNotExist(err) {
		t.Errorf("inv.yaml is still in the working directory (Stat error %v), want it moved to the archive", err)
	}
	list, err := svc.ListArchive()
	archived := list.Entries
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
