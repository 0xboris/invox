package opener_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

// The opener names the program in its errors: a *billing.ToolMissingError
// when it is not installed, a *billing.ToolFailedError when it fails.
func TestOpenReportsTheOpenersFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"not installed", fmt.Errorf("exec: %w", run.ErrNotFound), "xdg-open not found in PATH"},
		{"fails", &run.ExecError{Name: "xdg-open", Code: 4}, "xdg-open exited with status 4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			stub := runtest.NewStub(t)
			stub.Register("xdg-open", func(run.Cmd) error { return tc.err })

			err := opener.New(stub, ios, "linux").Open(context.Background(), "draft.eml")

			var missing *billing.ToolMissingError
			var failed *billing.ToolFailedError
			if !errors.As(err, &missing) && !errors.As(err, &failed) {
				t.Fatalf("Open error = %v, want a *billing.ToolMissingError or *billing.ToolFailedError", err)
			}
			if err.Error() != tc.want {
				t.Fatalf("Open error = %q, want %q", err, tc.want)
			}
		})
	}
}

func TestOpenRunsTheOSOpener(t *testing.T) {
	tests := []struct {
		goos string
		name string
		args []string
	}{
		{"darwin", "open", []string{"draft.eml"}},
		{"linux", "xdg-open", []string{"draft.eml"}},
		{"freebsd", "xdg-open", []string{"draft.eml"}},
		{"windows", "cmd", []string{"/c", "start", "", "draft.eml"}},
	}
	for _, tc := range tests {
		t.Run(tc.goos, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			stub := runtest.NewStub(t)
			var got run.Cmd
			stub.Register(tc.name, func(cmd run.Cmd) error {
				got = cmd
				fmt.Fprintln(cmd.Stdout, "opened")
				return nil
			})

			if err := opener.New(stub, ios, tc.goos).Open(context.Background(), "draft.eml"); err != nil {
				t.Fatalf("Open returned error: %v", err)
			}

			if !slices.Equal(got.Args, tc.args) {
				t.Fatalf("Args = %q, want %q", got.Args, tc.args)
			}
			if got.Stdout != ios.ErrOut || got.Stderr != ios.ErrOut {
				t.Fatalf("Stdout, Stderr = %v, %v, want ios.ErrOut itself", got.Stdout, got.Stderr)
			}
			if got.Stdin != nil {
				t.Fatalf("Stdin = %v, want none", got.Stdin)
			}
			if stdout.String() != "" || stderr.String() != "opened\n" {
				t.Fatalf("stdout, stderr = %q, %q, want \"\", %q", stdout.String(), stderr.String(), "opened\n")
			}
		})
	}
}
