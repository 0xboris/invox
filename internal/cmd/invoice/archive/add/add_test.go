package add

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
	"github.com/0xboris/invox/internal/iostreams"
)

// Each case runs as `archive add ARGS` and as `archive ARGS`, its deprecated
// form, which warns before it runs.
func TestNewCmdAddParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    AddOptions
		wantErr string
	}{
		{name: "positional", args: []string{"x.yaml"}, want: AddOptions{InvoicePath: "x.yaml"}},
		{name: "yes after the positional", args: []string{"x.yaml", "--yes"}, want: AddOptions{InvoicePath: "x.yaml", Yes: true}},
		{name: "dry run", args: []string{"-n", "x.yaml"}, want: AddOptions{InvoicePath: "x.yaml", DryRun: true}},
		{name: "input flag", args: []string{"-i", "x.yaml"}, want: AddOptions{InvoicePath: "x.yaml"}},
		{name: "input flag naming the positional", args: []string{"x.yaml", "-i", "./x.yaml"}, want: AddOptions{InvoicePath: "./x.yaml"}},
		{name: "input flag naming another file", args: []string{"F.yaml", "-i", "x.yaml"}, wantErr: "the INVOICE argument F.yaml and -i, --input x.yaml name different files; pass only one"},
		{name: "two positionals", args: []string{"x.yaml", "y.yaml"}, wantErr: "unexpected arguments: y.yaml"},
		{name: "no input", args: []string{}, wantErr: "missing required input: INVOICE.yaml or -i, --input"},
		{name: "misspelt flag", args: []string{"x.yaml", "--yse"}, wantErr: "unknown flag: --yse; did you mean --yes?"},
		{name: "after --", args: []string{"x.yaml", "--", "--yes"}, wantErr: "unexpected arguments: --yes"},
	}
	for _, form := range []struct {
		path        []string
		command     string
		wantWarning string
	}{
		{path: []string{"archive", "add"}, command: "archive add"},
		{path: []string{"archive"}, command: "archive", wantWarning: "warning: archive FILE is deprecated; use archive add FILE\n"},
	} {
		for _, tc := range tests {
			t.Run(form.command+"/"+tc.name, func(t *testing.T) {
				ios, _, _, errOut := iostreams.Test()
				f := cmdutil.NewFactory(ios, run.Exec{}, env.System())
				var got *AddOptions
				runF := func(_ context.Context, opts *AddOptions) error {
					got = opts
					return nil
				}
				archive := &cobra.Command{Use: "archive"}
				Configure(archive, f, runF, true)
				archive.AddCommand(NewCmdAdd(f, runF))
				root := &cobra.Command{Use: "invox"}
				root.AddCommand(archive)
				root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
				root.SetOut(io.Discard)
				root.SetErr(io.Discard)
				root.SetArgs(append(append([]string{}, form.path...), tc.args...))
				err := root.Execute()

				if tc.wantErr != "" {
					var flagErr *cmdutil.FlagError
					if !errors.As(err, &flagErr) || flagErr.Command != form.command || err.Error() != tc.wantErr || got != nil {
						t.Fatalf("Execute error = %#v, runF ran = %v; want FlagError for %s: %q and no run", err, got != nil, form.command, tc.wantErr)
					}
					if errOut.String() != "" {
						t.Errorf("stderr = %q, want nothing", errOut.String())
					}
					return
				}
				if err != nil {
					t.Fatalf("Execute returned error: %v", err)
				}
				if got == nil {
					t.Fatal("runF did not run")
				}
				if parsed := (AddOptions{InvoicePath: got.InvoicePath, Yes: got.Yes, DryRun: got.DryRun}); !reflect.DeepEqual(parsed, tc.want) {
					t.Errorf("parsed %+v, want %+v", parsed, tc.want)
				}
				if got.Command != form.command {
					t.Errorf("Command = %q, want %q", got.Command, form.command)
				}
				if errOut.String() != form.wantWarning {
					t.Errorf("stderr = %q, want %q", errOut.String(), form.wantWarning)
				}
			})
		}
	}
}
