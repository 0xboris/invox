package render

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// Relative paths are relative to the working directory invox was given,
// which need not be the process's own.
func TestRenderRunResolvesPathsAgainstGetwd(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		wantFile   string
	}{
		{name: "default output", wantFile: "invoice.tex"},
		{name: "relative -o", outputPath: filepath.Join("out", "x.tex"), wantFile: filepath.Join("out", "x.tex")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			host := invoice.NewHost(invoice.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
			if _, _, err := host.InitializeConfigDir(); err != nil {
				t.Fatal(err)
			}
			created, err := host.CreateNewInvoice(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), work, host.GlobalInvoiceDefaultsPath(), filepath.Join(work, "inv.yaml"), host.GlobalCustomersPath(), host.GlobalIssuerPath(), "CUST-001", invoice.NewInvoiceOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(work, "out"), 0o755); err != nil {
				t.Fatal(err)
			}

			ios, _, out, _ := iostreams.Test()
			opts := &RenderOptions{
				IO:          ios,
				Host:        func() invoice.Host { return host },
				Getwd:       func() (string, error) { return work, nil },
				InvoicePath: filepath.Base(created.Path),
				OutputPath:  tc.outputPath,
			}
			if err := renderRun(opts); err != nil {
				t.Fatalf("renderRun returned error: %v", err)
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
