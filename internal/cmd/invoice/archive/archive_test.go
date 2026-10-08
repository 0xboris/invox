package archive

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

func TestNewCmdArchiveParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    ArchiveOptions
		wantErr string
	}{
		{name: "positional", args: []string{"x.yaml"}, want: ArchiveOptions{InvoicePath: "x.yaml"}},
		{name: "yes after the positional", args: []string{"x.yaml", "--yes"}, want: ArchiveOptions{InvoicePath: "x.yaml", Yes: true}},
		{name: "input flag", args: []string{"-i", "x.yaml"}, want: ArchiveOptions{InvoicePath: "x.yaml"}},
		{name: "input flag after the positional", args: []string{"F.yaml", "-i", "x.yaml"}, wantErr: "unexpected arguments: F.yaml"},
		{name: "no input", args: []string{}, wantErr: "missing required input: INVOICE.yaml or -i, --input"},
		{name: "misspelt flag", args: []string{"x.yaml", "--yse"}, wantErr: "unknown flag: --yse; did you mean --yes?"},
		{name: "after --", args: []string{"x.yaml", "--", "--yes"}, wantErr: "unexpected arguments: --yes"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *ArchiveOptions
			cmd := NewCmdArchive(cmdutil.NewFactory(ios, run.Exec{}, env.System()), func(_ context.Context, opts *ArchiveOptions) error {
				got = opts
				return nil
			})
			root := &cobra.Command{Use: "invox"}
			root.AddCommand(cmd)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"archive"}, tc.args...))
			err := root.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || flagErr.Command != "archive" || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %#v, runF ran = %v; want FlagError for archive: %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil {
				t.Fatal("runF did not run")
			}
			if parsed := (ArchiveOptions{InvoicePath: got.InvoicePath, Yes: got.Yes}); !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
		})
	}
}
