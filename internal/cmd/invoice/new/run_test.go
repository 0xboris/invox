package newcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// The output path is relative to the working directory invox was given,
// which need not be the process's own.
func TestNewRunWritesOutputRelativeToGetwd(t *testing.T) {
	work := t.TempDir()
	host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
	if _, _, err := host.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}

	ios, _, out, _ := iostreams.Test()
	opts := &NewOptions{
		IO:         ios,
		Host:       func() invoice.Host { return host },
		Getwd:      func() (string, error) { return work, nil },
		Now:        func() time.Time { return time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC) },
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
