package edit

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

// archive edit copies into the working directory invox was given, which
// need not be the process's own.
func TestEditRunCopiesIntoGetwd(t *testing.T) {
	work := t.TempDir()
	draft := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"XDG_DATA_HOME": t.TempDir(), "INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}
	invoicePath := filepath.Join(draft, "inv.yaml")
	if _, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: draft, Output: invoicePath}); err != nil {
		t.Fatal(err)
	}
	if err := factorytest.SetStatus(invoicePath, invoice.Built); err != nil {
		t.Fatal(err)
	}
	result, err := svc.Archive(invoicePath, billing.ArchiveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &EditOptions{
		IO:       ios,
		Service:  f.Service,
		Getwd:    func() (string, error) { return work, nil },
		Filename: filepath.Base(result.Path),
	}
	if err := editRun(context.Background(), opts); err != nil {
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
