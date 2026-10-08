package email

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

func TestNewCmdEmailParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    EmailOptions
		wantErr string
	}{
		{name: "positional pdf", args: []string{"x.pdf"}, want: EmailOptions{InvoicePath: "x.pdf"}},
		{name: "every flag", args: []string{"x.yaml", "-p", "p.pdf", "-o", "d.eml", "--force", "-c", "c.yaml", "-u", "u.yaml", "--to", "a@b.example", "--subject", "Hi"}, want: EmailOptions{InvoicePath: "x.yaml", PDFPath: "p.pdf", OutputPath: "d.eml", Force: true, CustomersPath: "c.yaml", IssuerPath: "u.yaml", To: "a@b.example", Subject: "Hi"}},
		{name: "no input", args: []string{}, wantErr: "missing required input: INVOICE.yaml, INVOICE.pdf, or -i, --input"},
		{name: "bad input extension", args: []string{"x.txt"}, wantErr: "input must end with .yaml, .yml, or .pdf"},
		{name: "upper-case input extension", args: []string{"X.YAML"}, want: EmailOptions{InvoicePath: "X.YAML"}},
		{name: "bad pdf extension", args: []string{"x.yaml", "-p", "p.txt"}, wantErr: "-p, --pdf must end with .pdf"},
		{name: "bad output extension", args: []string{"x.yaml", "-o", "d.txt"}, wantErr: "-o, --output must end with .eml"},
		{name: "after --", args: []string{"x.yaml", "--", "--force"}, wantErr: "unexpected arguments: --force"},
		{name: "misspelt flag", args: []string{"x.yaml", "--forse"}, wantErr: "unknown flag: --forse; did you mean --force?"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var got *EmailOptions
			cmd := NewCmdEmail(cmdutil.NewFactory(ios, run.Exec{}, env.System()), func(_ context.Context, opts *EmailOptions) error {
				got = opts
				return nil
			})
			root := &cobra.Command{Use: "invox"}
			root.AddCommand(cmd)
			root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"email"}, tc.args...))
			err := root.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || flagErr.Command != "email" || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %#v, runF ran = %v; want FlagError for email: %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil {
				t.Fatal("runF did not run")
			}
			parsed := EmailOptions{InvoicePath: got.InvoicePath, PDFPath: got.PDFPath, OutputPath: got.OutputPath, CustomersPath: got.CustomersPath, IssuerPath: got.IssuerPath, To: got.To, Subject: got.Subject, Force: got.Force}
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
		})
	}
}
