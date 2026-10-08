package newcmd

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdNewParsing(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		want       NewOptions
		wantStderr string
		wantErr    string
	}{
		{name: "customer only", args: []string{"CUST-001"}, want: NewOptions{CustomerID: "CUST-001"}},
		{
			name: "every flag, after the positional",
			args: []string{" CUST-001 ", "-o", "out.yaml", "--defaults", "d.yaml", "-c", "c.yaml", "-u", "i.yaml", "-e", "--from-last"},
			want: NewOptions{CustomerID: "CUST-001", OutputPath: "out.yaml", DefaultsPath: "d.yaml", CustomersPath: "c.yaml", IssuerPath: "i.yaml", Edit: true, FromLast: true},
		},
		{
			name: "long flags",
			args: []string{"--output=out.yaml", "--defaults=d.yaml", "--customers", "c.yaml", "--issuer", "i.yaml", "--edit", "CUST-001"},
			want: NewOptions{CustomerID: "CUST-001", OutputPath: "out.yaml", DefaultsPath: "d.yaml", CustomersPath: "c.yaml", IssuerPath: "i.yaml", Edit: true},
		},
		{name: "deprecated -s", args: []string{"CUST-001", "-s", "d.yaml"}, want: NewOptions{CustomerID: "CUST-001", DefaultsPath: "d.yaml"}, wantStderr: "warning: -s, --source is deprecated; use --defaults\n"},
		{name: "deprecated --source", args: []string{"--source", "d.yaml", "CUST-001"}, want: NewOptions{CustomerID: "CUST-001", DefaultsPath: "d.yaml"}, wantStderr: "warning: -s, --source is deprecated; use --defaults\n"},
		{name: "missing customer", args: []string{}, wantErr: "missing required arguments: CUSTOMER_ID"},
		{name: "dangling output", args: []string{"CUST-001", "-o"}, wantErr: "flag needs an argument: -o"},
		{name: "misspelt flag", args: []string{"CUST-001", "--form-last"}, wantErr: "unknown flag: --form-last; did you mean --from-last?"},
		{name: "flag after --", args: []string{"CUST-001", "--", "--from-last"}, wantErr: "unexpected arguments: --from-last"},
		{name: "wrong output extension", args: []string{"CUST-001", "-o", "out.yml"}, wantErr: "-o, --output must end with .yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, errOut := iostreams.Test()
			var got *NewOptions
			cmd := NewCmdNew(factory.New(ios, run.Exec{}, env.System()), func(_ context.Context, opts *NewOptions) error {
				got = opts
				return nil
			})
			cmd.SetFlagErrorFunc(cmdutil.FlagErrorFunc)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr || got != nil {
					t.Fatalf("Execute error = %v, runF ran = %v; want FlagError %q and no run", err, got != nil, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil {
				t.Fatal("runF did not run")
			}
			parsed := NewOptions{CustomerID: got.CustomerID, OutputPath: got.OutputPath, DefaultsPath: got.DefaultsPath, CustomersPath: got.CustomersPath, IssuerPath: got.IssuerPath, FromLast: got.FromLast, Edit: got.Edit}
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
			if errOut.String() != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", errOut.String(), tc.wantStderr)
			}
		})
	}
}
