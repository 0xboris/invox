package list

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

func TestNewCmdListParsing(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		wantErr string
	}{
		{args: []string{}},
		{args: []string{"x"}, wantErr: "unexpected arguments: x"},
		{args: []string{"--bogus"}, wantErr: "unknown flag: --bogus"},
	} {
		ios, _, _, _ := iostreams.Test()
		ran := false
		root := &cobra.Command{Use: "invox"}
		archive := &cobra.Command{Use: "archive"}
		archive.AddCommand(NewCmdList(factory.New(ios, run.Exec{}, env.System()), func(*ListOptions) error {
			ran = true
			return nil
		}))
		root.AddCommand(archive)
		root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs(append([]string{"archive", "list"}, tc.args...))
		err := root.Execute()

		if tc.wantErr == "" {
			if err != nil || !ran {
				t.Errorf("%q: Execute() = %v, runF ran = %v; want nil, true", tc.args, err, ran)
			}
			continue
		}
		var flagErr *cmdutil.FlagError
		if !errors.As(err, &flagErr) || flagErr.Command != "archive list" || err.Error() != tc.wantErr || ran {
			t.Errorf("%q: Execute error = %#v, runF ran = %v; want FlagError for archive list: %q and no run", tc.args, err, ran, tc.wantErr)
		}
	}
}
