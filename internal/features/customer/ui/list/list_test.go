package list

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/adapters/fsutil"
	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/adapters/starter"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/customer/ui/shared"
	"github.com/0xboris/invox/internal/features/customer/usecases"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

func TestNewCmdListParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantCustomers string
		wantErr       string
	}{
		{name: "no flags", args: []string{}},
		{name: "shorthand", args: []string{"-c", "c.yaml"}, wantCustomers: "c.yaml"},
		{name: "long with equals", args: []string{"--customers=c.yaml"}, wantCustomers: "c.yaml"},
		{name: "extra argument", args: []string{"extra"}, wantErr: "unexpected arguments: extra"},
		{name: "dangling flag", args: []string{"-c"}, wantErr: "flag needs an argument: -c"},
		{name: "misspelt flag", args: []string{"--customer", "c.yaml"}, wantErr: "unknown flag: --customer; did you mean --customers?"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *ListOptions
			cmd := NewCmdList(&shared.Factory{IOStreams: ios, Env: platform.Env(), Hosts: invoice.NewHostSource(func(string) invoice.Host { return invoice.Host{} })}, func(opts *ListOptions) error {
				got = opts
				return nil
			})
			cmd.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
					t.Fatalf("Execute error = %v, want FlagError %q", err, tc.wantErr)
				}
				if got != nil {
					t.Error("runF ran after a usage error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil || got.CustomersPath != tc.wantCustomers {
				t.Fatalf("opts = %+v, want CustomersPath %q", got, tc.wantCustomers)
			}
		})
	}
}

func TestListRun(t *testing.T) {
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "c.yaml"), []byte("B-1:\n  name: Beta\n  status: inactive\nA-1:\n  name: Alpha\n  status: active\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	host := invoice.NewHost(invoice.HostInputs{Files: fsutil.Files{}, Starter: starter.Files, GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})

	ios, _, out, errOut := iostreams.Test()
	opts := &ListOptions{
		IO:            ios,
		ListCustomers: usecases.NewListCustomers(invoice.NewHostSource(func(string) invoice.Host { return host })),
		Host:          func() invoice.Host { return host },
		Getwd:         func() (string, error) { return work, nil },
		CustomersPath: "c.yaml",
	}
	if err := listRun(opts); err != nil {
		t.Fatalf("listRun returned error: %v", err)
	}
	if want := "A-1\tAlpha\tactive\nB-1\tBeta\tinactive\n"; out.String() != want {
		t.Errorf("stdout = %q, want %q", out.String(), want)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr = %q, want empty", errOut.String())
	}
}
