package cmdutil

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestFlagErrorFuncNamesTheShorthand(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"-c"}, want: "flag needs an argument: -c"},
		{args: []string{"-x"}, want: "unknown shorthand flag: -x"},
		{args: []string{"-vx"}, want: "unknown shorthand flag: -x"},
	}
	for _, tc := range tests {
		root := &cobra.Command{Use: "invox"}
		cmd := &cobra.Command{Use: "list", RunE: func(*cobra.Command, []string) error { return nil }}
		cmd.Flags().StringP("customers", "c", "", "")
		cmd.Flags().BoolP("verbose", "v", false, "")
		root.AddCommand(cmd)
		root.SetFlagErrorFunc(FlagErrorFunc)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs(append([]string{"list"}, tc.args...))

		err := root.Execute()
		var flagErr *FlagError
		if !errors.As(err, &flagErr) || flagErr.Command != "list" || err.Error() != tc.want {
			t.Errorf("%q: error = %#v, want FlagError for list: %q", tc.args, err, tc.want)
		}
	}
}
