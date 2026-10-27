package config

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/config/ui/edit"
	"github.com/0xboris/invox/internal/features/config/ui/shared"
	"github.com/0xboris/invox/internal/features/config/usecases"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

func TestNewCmdConfigParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantRun bool
		wantErr string
	}{
		{name: "no arguments", args: []string{"config"}, wantRun: true},
		{name: "extra argument", args: []string{"config", "extra"}, wantErr: "unexpected arguments: extra"},
		{name: "edit", args: []string{"config", "edit"}, wantRun: true},
		{name: "edit extra argument", args: []string{"config", "edit", "extra"}, wantErr: "unexpected arguments: extra"},
		{name: "paths extra argument", args: []string{"config", "paths", "extra"}, wantErr: "unexpected arguments: extra"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ran := false
			root := &cobra.Command{Use: "invox", SilenceErrors: true, SilenceUsage: true}
			root.AddCommand(NewCmdConfig(&shared.Factory{IOStreams: ios, Env: platform.Env(), Hosts: invoice.NewHostSource(func(string) invoice.Host { return invoice.Host{} }), EditConfig: usecases.NewEditConfig(stubEditor{})}, func(_ context.Context, opts *edit.EditOptions) error {
				ran = opts.EditConfig == usecases.NewEditConfig(stubEditor{})
				return nil
			}))
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(tc.args)
			err := root.Execute()

			if ran != tc.wantRun {
				t.Errorf("runF ran with an editor = %v, want %v", ran, tc.wantRun)
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Execute returned error: %v", err)
				}
				return
			}
			var flagErr *cmdutil.FlagError
			if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
				t.Fatalf("Execute error = %v, want FlagError %q", err, tc.wantErr)
			}
		})
	}
}

// stubEditor is an editor that never opens.
type stubEditor struct{}

func (stubEditor) Edit(context.Context, string) error { return nil }
