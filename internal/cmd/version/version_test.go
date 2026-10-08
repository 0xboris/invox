package version

import (
	"errors"
	"io"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdVersionParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantRun bool
		wantErr string
	}{
		{name: "no arguments", args: []string{}, wantRun: true},
		{name: "extra arguments", args: []string{"a", "b"}, wantErr: "unexpected arguments for version: a b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := factory.New(ios, run.Exec{}, env.System())
			ran := false
			cmd := NewCmdVersion(f, func(opts *VersionOptions) error {
				ran = true
				if opts.IO != ios {
					t.Error("opts.IO is not the factory's streams")
				}
				return nil
			})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()

			if ran != tc.wantRun {
				t.Errorf("runF called = %v, want %v", ran, tc.wantRun)
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Execute returned error: %v", err)
				}
				return
			}
			var flagErr *cmdutil.FlagError
			if !errors.As(err, &flagErr) || flagErr.Command != "version" || err.Error() != tc.wantErr {
				t.Fatalf("Execute error = %#v, want FlagError for version: %q", err, tc.wantErr)
			}
		})
	}
}

func TestVersionRun(t *testing.T) {
	ios, _, out, errOut := iostreams.Test()
	if err := versionRun(&VersionOptions{IO: ios}); err != nil {
		t.Fatalf("versionRun returned error: %v", err)
	}
	if got := out.String(); got != "invox version DEV\n" {
		t.Errorf("stdout = %q, want %q", got, "invox version DEV\n")
	}
	if got := errOut.String(); got != "" {
		t.Errorf("stderr = %q, want empty", got)
	}
}
