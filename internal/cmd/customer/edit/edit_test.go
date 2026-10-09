package edit

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, errOut := iostreams.Test()
			var got *EditOptions
			cmd := NewCmdEdit(factory.New(ios, run.Exec{}, env.System()), func(_ context.Context, opts *EditOptions) error {
				got = opts
				return nil
			})
			cmd.SetOut(io.Discard)
			cmd.SetErr(ios.ErrOut)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
					t.Fatalf("Execute error = %#v, want FlagError %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil || got.CustomersPath != tc.wantCustomers || got.Editor == nil {
				t.Fatalf("opts = %+v, want CustomersPath %q and an editor", got, tc.wantCustomers)
			}
			if errOut.String() != "" {
				t.Errorf("stderr = %q, want empty", errOut.String())
			}
		})
	}
}
