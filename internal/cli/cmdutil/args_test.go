package cmdutil

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestPositionalArgs(t *testing.T) {
	tests := []struct {
		name    string
		check   cobra.PositionalArgs
		args    []string
		wantErr string
	}{
		{name: "no args", check: NoArgs},
		{name: "unexpected", check: NoArgs, args: []string{"a", "b"}, wantErr: "unexpected arguments: a b"},
		{name: "exact", check: ExactArgs("FROM", "TO"), args: []string{"a", "b"}},
		{name: "all missing", check: ExactArgs("FROM", "TO"), wantErr: "missing required arguments: FROM TO"},
		{name: "one missing", check: ExactArgs("FROM", "TO"), args: []string{"a"}, wantErr: "missing required arguments: TO"},
		{name: "one too many", check: ExactArgs("FROM", "TO"), args: []string{"a", "b", "c"}, wantErr: "unexpected arguments: c"},
		{name: "up to one", check: MaximumArgs(1), args: []string{"a"}},
		{name: "past one", check: MaximumArgs(1), args: []string{"a", "b", "c"}, wantErr: "unexpected arguments: b c"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.check(&cobra.Command{}, tc.args)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
				return
			}
			var flagErr *FlagError
			if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
				t.Fatalf("error = %#v, want FlagError %q", err, tc.wantErr)
			}
		})
	}
}
