package email

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// The invoice, its PDF and -o are relative to the working directory invox
// was given, which need not be the process's own.
func TestEmailRunResolvesPathsAgainstGetwd(t *testing.T) {
	work := t.TempDir()
	stub := runtest.NewStub(t)
	var opened string
	stub.Register("xdg-open", func(cmd run.Cmd) error {
		opened = cmd.Args[0]
		return nil
	})
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }, Runner: stub})
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
	if err := os.WriteFile(filepath.Join(work, "inv.pdf"), []byte("%PDF\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &EmailOptions{
		IO:          ios,
		Service:     f.Service,
		Getwd:       func() (string, error) { return work, nil },
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

// An invoice that isn't built yet fails before the mail app is asked to
// compose anything.
func TestEmailRunDraftInvoiceNeverComposes(t *testing.T) {
	work := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "darwin", Vars: map[string]string{"INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: work, Output: filepath.Join(work, "inv.yaml")}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "inv.pdf"), []byte("%PDF\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The Factory's runner has nothing registered, so a Compose call would
	// fail the test.
	ios, _, out, _ := iostreams.Test()
	opts := &EmailOptions{
		IO:          ios,
		Service:     f.Service,
		Getwd:       func() (string, error) { return work, nil },
		InvoicePath: "inv.yaml",
	}
	err := emailRun(context.Background(), opts, false)
	const want = "invoice.status must be `built` or `archived` before creating an email draft, got `draft`"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("emailRun error = %v, want one containing %q", err, want)
	}
	if errors.As(err, new(*cmdutil.FlagError)) {
		t.Errorf("emailRun error is a usage error, want a runtime error (exit 1)")
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}
