package newcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
)

// The output path is relative to the working directory invox was given,
// which need not be the process's own.
func TestNewRunWritesOutputRelativeToGetwd(t *testing.T) {
	work := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &NewOptions{
		IO:         ios,
		Service:    f.Service,
		Getwd:      func() (string, error) { return work, nil },
		CustomerID: "CUST-001",
		OutputPath: "out.yaml",
	}
	if err := newRun(context.Background(), opts); err != nil {
		t.Fatalf("newRun returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "out.yaml")); err != nil {
		t.Errorf("out.yaml not written to the working directory: %v", err)
	}
	if got := out.String(); got != "out.yaml\n" {
		t.Errorf("stdout = %q, want %q", got, "out.yaml\n")
	}
}

// Without --from-last, a missing invoice_defaults.yaml is a usage error that
// names the ways to provide one.
func TestNewRunWithoutDefaultsIsAUsageError(t *testing.T) {
	work := t.TempDir()
	configDir := t.TempDir()
	f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"INVOX_CONFIG_DIR": configDir}, Getwd: func() (string, error) { return work, nil }})
	svc := f.Service(cmdutil.Files{})
	if _, err := svc.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(configDir, "invoice_defaults.yaml")); err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &NewOptions{
		IO:         ios,
		Service:    f.Service,
		Getwd:      func() (string, error) { return work, nil },
		CustomerID: "CUST-001",
	}
	err := newRun(context.Background(), opts)

	var flagErr *cmdutil.FlagError
	want := "defaults file not found; pass --defaults, set paths.defaults in config.yaml, or place invoice_defaults.yaml at " + filepath.Join(configDir, "invoice_defaults.yaml")
	if !errors.As(err, &flagErr) || err.Error() != want {
		t.Fatalf("newRun error = %#v, want FlagError %q", err, want)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}
