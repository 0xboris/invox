package render

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
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
			f := factorytest.New(t, nil, factorytest.Options{GOOS: "linux", Vars: map[string]string{"INVOX_CONFIG_DIR": t.TempDir()}, Getwd: func() (string, error) { return work, nil }})
			svc := f.Service(cmdutil.Files{})
			if _, err := svc.Init(); err != nil {
				t.Fatal(err)
			}
			created, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: work, Output: filepath.Join(work, "inv.yaml")})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(work, "out"), 0o755); err != nil {
				t.Fatal(err)
			}

			ios, _, out, _ := iostreams.Test()
			opts := &RenderOptions{
				IO:          ios,
				Service:     f.Service,
				Getwd:       func() (string, error) { return work, nil },
				InvoicePath: filepath.Base(created.Path),
				OutputPath:  tc.outputPath,
			}
			if err := renderRun(context.Background(), opts); err != nil {
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
