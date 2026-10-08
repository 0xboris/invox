package validate

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdValidateParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    ValidateOptions
		wantErr string
	}{
		{name: "short", args: []string{"-i", "x.yaml", "-c", "c.yaml", "-u", "u.yaml"}, want: ValidateOptions{InvoicePath: "x.yaml", CustomersPath: "c.yaml", IssuerPath: "u.yaml"}},
		{name: "long", args: []string{"--input", "x.yaml", "--customers=c.yaml", "--issuer", "u.yaml"}, want: ValidateOptions{InvoicePath: "x.yaml", CustomersPath: "c.yaml", IssuerPath: "u.yaml"}},
		{name: "misspelt archive", args: []string{"--archiv"}, wantErr: "unknown flag: --archiv"},
		{name: "no input", args: []string{}, wantErr: "missing required flags: -i, --input"},
		{name: "positional input", args: []string{"x.yaml"}, wantErr: "unexpected arguments: x.yaml"},
		{name: "misspelt flag", args: []string{"-i", "x.yaml", "--inptu", "y"}, wantErr: "unknown flag: --inptu; did you mean --input?"},
		{name: "dangling input", args: []string{"-i"}, wantErr: "flag needs an argument: -i"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *ValidateOptions
			cmd := NewCmdValidate(cmdutil.NewFactory(ios, run.Exec{}, env.System()), func(opts *ValidateOptions) error {
				got = opts
				return nil
			})
			root := &cobra.Command{Use: "invox"}
			root.AddCommand(cmd)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"validate"}, tc.args...))
			err := root.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || flagErr.Command != "validate" || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %#v, runF ran = %v; want FlagError for validate: %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil {
				t.Fatal("runF did not run")
			}
			parsed := ValidateOptions{InvoicePath: got.InvoicePath, CustomersPath: got.CustomersPath, IssuerPath: got.IssuerPath}
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
		})
	}
}
