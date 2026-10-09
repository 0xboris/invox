package edit

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdEditParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantFile string
		wantErr  string
	}{
		{name: "filename", args: []string{" 2026-03-06.yaml "}, wantFile: "2026-03-06.yaml"},
		{name: "missing filename", args: []string{}, wantErr: "missing required arguments: FILENAME"},
		{name: "two filenames", args: []string{"a.yaml", "b.yaml"}, wantErr: "unexpected arguments: b.yaml"},
		{name: "unknown flag", args: []string{"a.yaml", "--bogus"}, wantErr: "unknown flag: --bogus"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *EditOptions
			root := &cobra.Command{Use: "invox"}
			archive := &cobra.Command{Use: "archive"}
			archive.AddCommand(NewCmdEdit(factory.New(ios, run.Exec{}, env.System()), func(opts *EditOptions) error {
				got = opts
				return nil
			}))
			root.AddCommand(archive)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"archive", "edit"}, tc.args...))
			err := root.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %#v, runF ran = %v; want FlagError %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil || got.Filename != tc.wantFile {
				t.Fatalf("opts = %+v, want Filename %q", got, tc.wantFile)
			}
		})
	}
}
