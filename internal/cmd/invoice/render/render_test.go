package render

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

func TestNewCmdRenderParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    RenderOptions
		wantErr string
	}{
		{name: "short", args: []string{"-i", "x.yaml", "-o", "o.tex", "-c", "c.yaml", "-u", "u.yaml", "-t", "plain"}, want: RenderOptions{InvoicePath: "x.yaml", OutputPath: "o.tex", Support: cmdutil.SupportPaths{Customers: "c.yaml", Issuer: "u.yaml", Template: "plain"}}},
		{name: "long", args: []string{"--input", "x.yaml", "--output", "o.tex", "--customers", "c.yaml", "--issuer", "u.yaml", "--template", "t.tex"}, want: RenderOptions{InvoicePath: "x.yaml", OutputPath: "o.tex", Support: cmdutil.SupportPaths{Customers: "c.yaml", Issuer: "u.yaml", Template: "t.tex"}}},
		{name: "wrong output extension", args: []string{"-i", "x.yaml", "-o", "o.pdf"}, wantErr: "-o, --output must end with .tex"},
		{name: "upper-case output extension", args: []string{"-i", "x.yaml", "-o", "OUT.TEX"}, wantErr: "-o, --output must end with .tex"},
		{name: "blank output", args: []string{"-i", "x.yaml", "-o", " "}, want: RenderOptions{InvoicePath: "x.yaml", OutputPath: " "}},
		{name: "no input", args: []string{}, wantErr: "missing required input: INVOICE.yaml or -i, --input"},
		{name: "positional input", args: []string{"x.yaml"}, want: RenderOptions{InvoicePath: "x.yaml"}},
		{name: "positional and input naming it", args: []string{"x.yaml", "-i", "x.yaml"}, want: RenderOptions{InvoicePath: "x.yaml"}},
		{name: "positional and input naming another file", args: []string{"x.yaml", "-i", "y.yaml"}, wantErr: "the INVOICE argument x.yaml and -i, --input y.yaml name different files; pass only one"},
		{name: "two positionals", args: []string{"x.yaml", "y.yaml"}, wantErr: "unexpected arguments: y.yaml"},
		{name: "misspelt flag", args: []string{"-i", "x.yaml", "--inptu", "y"}, wantErr: "unknown flag: --inptu; did you mean --input?"},
		{name: "dangling input", args: []string{"-i"}, wantErr: "flag needs an argument: -i"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *RenderOptions
			cmd := NewCmdRender(factory.New(ios, run.Exec{}, env.System()), func(_ context.Context, opts *RenderOptions) error {
				got = opts
				return nil
			})
			root := &cobra.Command{Use: "invox"}
			root.AddCommand(cmd)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"render"}, tc.args...))
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
			parsed := RenderOptions{InvoicePath: got.InvoicePath, OutputPath: got.OutputPath, Support: got.Support}
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
		})
	}
}
