package template

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/platform"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/template/ui/shared"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
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
			root.AddCommand(NewCmdTemplate(&shared.Factory{IOStreams: ios, Env: platform.Env(), Hosts: invoice.NewHostSource(func(string) invoice.Host { return invoice.Host{} })}))
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
