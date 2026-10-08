package initcmd

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdInitParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantForce bool
		wantErr   string
	}{
		{name: "no flags", args: []string{}},
		{name: "force", args: []string{"--force"}, wantForce: true},
		{name: "extra argument", args: []string{"extra"}, wantErr: "unexpected arguments: extra"},
		{name: "misspelt flag", args: []string{"--forse"}, wantErr: "unknown flag: --forse; did you mean --force?"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *InitOptions
			cmd := NewCmdInit(cmdutil.NewFactory(ios, run.Exec{}, env.System()), func(_ context.Context, opts *InitOptions) error {
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
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %v, runF ran = %v; want FlagError %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil || got.Force != tc.wantForce {
				t.Fatalf("opts = %+v, want Force %v", got, tc.wantForce)
			}
		})
	}
}
