package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// Relative paths are relative to the working directory invox was given,
// which need not be the process's own.
func TestBuildRunResolvesPathsAgainstGetwd(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		wantFile   string
	}{
		{name: "default output", wantFile: "inv.pdf"},
		{name: "relative -o", outputPath: filepath.Join("out", "x.pdf"), wantFile: filepath.Join("out", "x.pdf")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
			if _, _, err := host.InitializeConfigDir(); err != nil {
				t.Fatal(err)
			}
			if _, err := host.CreateNewInvoice(invoice.NewInvoiceParams{Now: time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), WorkDir: work, DefaultsPath: host.GlobalInvoiceDefaultsPath(), OutputPath: filepath.Join(work, "inv.yaml"), CustomersPath: host.GlobalCustomersPath(), IssuerPath: host.GlobalIssuerPath(), CustomerID: "CUST-001"}); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(work, "out"), 0o755); err != nil {
				t.Fatal(err)
			}

			ios, _, out, _ := iostreams.Test()
			stub := run.NewStub(t)
			stub.Register("tectonic", func(cmd run.Cmd) error {
				pdf := strings.TrimSuffix(cmd.Args[0], ".tex") + ".pdf"
				return os.WriteFile(filepath.Join(cmd.Dir, pdf), []byte("%PDF\n"), 0o644)
			})
			opts := &BuildOptions{
				IO:          ios,
				Compiler:    tectonic.New(stub, ios, "linux"),
				Host:        func() invoice.Host { return host },
				Getwd:       func() (string, error) { return work, nil },
				Now:         time.Now,
				InvoicePath: "inv.yaml",
				OutputPath:  tc.outputPath,
			}
			if err := buildRun(context.Background(), opts); err != nil {
				t.Fatalf("buildRun returned error: %v", err)
			}
			if _, err := os.Stat(filepath.Join(work, tc.wantFile)); err != nil {
				t.Errorf("%s not written to the working directory: %v", tc.wantFile, err)
			}
			if got := out.String(); got != tc.wantFile+"\n" {
				t.Errorf("stdout = %q, want %q", got, tc.wantFile+"\n")
			}
		})
	}
}
