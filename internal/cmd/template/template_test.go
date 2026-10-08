package template

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdTemplateUnknownSubcommand(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{arg: "lsit", want: `unknown template subcommand "lsit"; did you mean "list"?`},
		{arg: "li", want: `unknown template subcommand "li"; did you mean "list"?`},
		{arg: "bogus", want: `unknown template subcommand "bogus"`},
	}
	for _, tc := range tests {
		t.Run(tc.arg, func(t *testing.T) {
			ios, _, out, _ := iostreams.Test()
			root := &cobra.Command{Use: "invox", SilenceErrors: true, SilenceUsage: true}
			root.AddCommand(NewCmdTemplate(cmdutil.NewFactory(ios, run.Exec{}, env.System())))
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs([]string{"template", tc.arg})
			err := root.Execute()

			var flagErr *cmdutil.FlagError
			if !errors.As(err, &flagErr) || flagErr.Command != "template" || err.Error() != tc.want {
				t.Fatalf("Execute error = %#v, want FlagError for template: %q", err, tc.want)
			}
			if out.Len() != 0 {
				t.Errorf("stdout = %q, want empty", out.String())
			}
		})
	}
}
