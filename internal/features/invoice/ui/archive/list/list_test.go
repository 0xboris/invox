package list

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/invoice/ui/shared"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
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
		archive.AddCommand(NewCmdList(&shared.Factory{IOStreams: ios, Env: platform.Env(), Hosts: invoice.NewHostSource(func(string) invoice.Host { return invoice.Host{} })}, func(*ListOptions) error {
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
