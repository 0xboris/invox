package build

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdBuildParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    BuildOptions
		wantErr string
	}{
		{name: "positional input", args: []string{"x.yaml"}, want: BuildOptions{InvoicePath: "x.yaml"}},
		{name: "flags after the positional", args: []string{"x.yaml", "-o", "o.pdf", "-c", "c.yaml", "-u", "u.yaml", "-t", "plain", "--archive", "--yes"}, want: BuildOptions{InvoicePath: "x.yaml", OutputPath: "o.pdf", CustomersPath: "c.yaml", IssuerPath: "u.yaml", TemplatePath: "plain", Archive: true, Yes: true}},
		{name: "input flag", args: []string{"--input", "x.yaml"}, want: BuildOptions{InvoicePath: "x.yaml"}},
		{name: "input flag and positional", args: []string{"-i", "x.yaml", "y.yaml"}, wantErr: "the INVOICE argument y.yaml and -i, --input x.yaml name different files; pass only one"},
		{name: "input flag naming the positional", args: []string{"-i", "x.yaml", "x.yaml"}, want: BuildOptions{InvoicePath: "x.yaml"}},
		{name: "two positionals", args: []string{"x.yaml", "y.yaml"}, wantErr: "unexpected arguments: y.yaml"},
		{name: "no input", args: []string{}, wantErr: "missing required input: INVOICE.yaml or -i, --input"},
		{name: "dangling output", args: []string{"x.yaml", "-o"}, wantErr: "flag needs an argument: -o"},
		{name: "misspelt flag", args: []string{"x.yaml", "--archiv"}, wantErr: "unknown flag: --archiv; did you mean --archive?"},
		{name: "wrong output extension", args: []string{"x.yaml", "-o", "o.tex"}, wantErr: "-o, --output must end with .pdf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *BuildOptions
			cmd := NewCmdBuild(factory.New(ios, run.Exec{}, env.System()), func(_ context.Context, opts *BuildOptions) error {
				got = opts
				return nil
			})
			root := &cobra.Command{Use: "invox"}
			root.AddCommand(cmd)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"build"}, tc.args...))
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
			if got == nil {
				t.Fatal("runF did not run")
			}
			parsed := BuildOptions{InvoicePath: got.InvoicePath, OutputPath: got.OutputPath, CustomersPath: got.CustomersPath, IssuerPath: got.IssuerPath, TemplatePath: got.TemplatePath, Archive: got.Archive, Yes: got.Yes}
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
		})
	}
}
