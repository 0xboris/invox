package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
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
			ios, _, out, _ := iostreams.Test()
			stub := run.NewStub(t)
			stub.Register("tectonic", func(cmd run.Cmd) error {
				pdf := strings.TrimSuffix(cmd.Args[0], ".tex") + ".pdf"
				return os.WriteFile(filepath.Join(cmd.Dir, pdf), []byte("%PDF\n"), 0o644)
			})
			f := factorytest.New(t, ios, factorytest.Options{
				Vars:   map[string]string{"INVOX_CONFIG_DIR": t.TempDir()},
				Getwd:  func() (string, error) { return work, nil },
				Runner: stub,
			})
			svc := f.Service(cmdutil.Files{})
			if _, err := svc.Init(); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.New(billing.NewRequest{CustomerID: "CUST-001", WorkDir: work, Output: filepath.Join(work, "inv.yaml")}); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(work, "out"), 0o755); err != nil {
				t.Fatal(err)
			}

			opts := &BuildOptions{
				IO:          ios,
				Service:     f.Service,
				Getwd:       func() (string, error) { return work, nil },
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
