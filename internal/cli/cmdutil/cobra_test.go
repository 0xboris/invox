package cmdutil

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestFlagErrorFuncInvalidValue(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--names=bogus"}, want: `invalid value "bogus" for --names`},
		{args: []string{"--limit", "x"}, want: `invalid value "x" for --limit`},
		{args: []string{"-l", "x"}, want: `invalid value "x" for --limit`},
	}
	for _, tc := range tests {
		root := &cobra.Command{Use: "invox"}
		cmd := &cobra.Command{Use: "list", RunE: func(*cobra.Command, []string) error { return nil }}
		cmd.Flags().Bool("names", false, "")
		cmd.Flags().IntP("limit", "l", 0, "")
		root.AddCommand(cmd)
		root.SetFlagErrorFunc(FlagErrorFunc)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs(append([]string{"list"}, tc.args...))

		err := root.Execute()
		var flagErr *FlagError
		if !errors.As(err, &flagErr) || err.Error() != tc.want {
			t.Errorf("%q: error = %#v, want FlagError %q", tc.args, err, tc.want)
		}
	}
}
