package edit

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/customer/ui/shared"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// Each case runs as `customer edit` and as `customer config`, its
// deprecated name, which warns before it runs.
func TestNewCmdEditParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantCustomers string
		wantErr       string
	}{
		{name: "no flags", args: []string{}},
		{name: "shorthand", args: []string{"-c", "c.yaml"}, wantCustomers: "c.yaml"},
		{name: "long", args: []string{"--customers", "c.yaml"}, wantCustomers: "c.yaml"},
		{name: "extra argument", args: []string{"extra"}, wantErr: "unexpected arguments: extra"},
	}
	for _, form := range []struct {
		name        string
		newCmd      func(*shared.Factory, func(context.Context, *EditOptions) error) *cobra.Command
		wantWarning string
	}{
		{name: "edit", newCmd: NewCmdEdit},
		{name: "config", newCmd: NewCmdConfig, wantWarning: "warning: customer config is deprecated; use customer edit\n"},
	} {
		for _, tc := range tests {
			t.Run(form.name+"/"+tc.name, func(t *testing.T) {
				ios, _, _, errOut := iostreams.Test()
				var got *EditOptions
				cmd := form.newCmd(&shared.Factory{IOStreams: ios, Env: platform.Env(), Hosts: invoice.NewHostSource(func(string) invoice.Host { return invoice.Host{} }), Editor: stubEditor{}}, func(_ context.Context, opts *EditOptions) error {
					got = opts
					return nil
				})
				cmd.SetOut(io.Discard)
				cmd.SetErr(ios.ErrOut)
				cmd.SetArgs(tc.args)
				err := cmd.Execute()

				command := "customer " + form.name
				if tc.wantErr != "" {
					var flagErr *cmdutil.FlagError
					if !errors.As(err, &flagErr) || flagErr.Command != command || err.Error() != tc.wantErr {
						t.Fatalf("Execute error = %#v, want FlagError for %s: %q", err, command, tc.wantErr)
					}
					return
				}
				if err != nil {
					t.Fatalf("Execute returned error: %v", err)
				}
				if got == nil || got.CustomersPath != tc.wantCustomers || got.Editor == nil || got.Command != command {
					t.Fatalf("opts = %+v, want CustomersPath %q, an editor and Command %q", got, tc.wantCustomers, command)
				}
				if errOut.String() != form.wantWarning {
					t.Errorf("stderr = %q, want %q", errOut.String(), form.wantWarning)
				}
			})
		}
	}
}

// stubEditor is an editor that never opens.
type stubEditor struct{}

func (stubEditor) Edit(context.Context, string) error { return nil }
