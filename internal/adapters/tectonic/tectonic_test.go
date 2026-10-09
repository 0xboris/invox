package tectonic_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestBuildRunsTectonicInTheFilesDirectory(t *testing.T) {
	ios, stdin, stdout, stderr := iostreams.Test()
	stdin.WriteString("answer\n")
	stub := runtest.NewStub(t)
	var got run.Cmd
	var input []byte
	stub.Register("tectonic", func(cmd run.Cmd) error {
		got = cmd
		input, _ = io.ReadAll(cmd.Stdin)
		fmt.Fprintln(cmd.Stdout, "note: writing invoice.pdf")
		fmt.Fprintln(cmd.Stderr, "warning: overfull hbox")
		return nil
	})
	texPath := filepath.Join("build", "tmp", "invoice.tex")

	if err := tectonic.New(stub, ios, "linux").Build(context.Background(), texPath); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if want := filepath.Join("build", "tmp"); got.Dir != want {
		t.Fatalf("Dir = %q, want %q", got.Dir, want)
	}
	if want := []string{"invoice.tex"}; !slices.Equal(got.Args, want) {
		t.Fatalf("Args = %q, want %q", got.Args, want)
	}
	if got.Stdin != ios.In || got.Stdout != ios.ErrOut || got.Stderr != ios.ErrOut {
		t.Fatalf("streams = (%v, %v, %v), want ios.In, ios.ErrOut, ios.ErrOut themselves", got.Stdin, got.Stdout, got.Stderr)
	}
	if string(input) != "answer\n" {
		t.Fatalf("stdin = %q, want %q", input, "answer\n")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if want := "note: writing invoice.pdf\nwarning: overfull hbox\n"; stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestBuildReturnsExecErrorWhenTectonicFails(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	stub := runtest.NewStub(t)
	stub.Register("tectonic", func(run.Cmd) error { return &run.ExecError{Name: "tectonic", Code: 1} })

	err := tectonic.New(stub, ios, "linux").Build(context.Background(), "invoice.tex")

	var execErr *run.ExecError
	if !errors.As(err, &execErr) || execErr.Code != 1 {
		t.Fatalf("Build error = %v, want *run.ExecError with Code 1", err)
	}
}

func TestBuildReportsMissingTectonicWithAnInstallHint(t *testing.T) {
	tests := []struct {
		goos string
		hint string
	}{
		{"darwin", "Install it with 'brew install tectonic', then rerun this command."},
		{"linux", "Install it from https://tectonic-typesetting.github.io, then rerun this command."},
		{"windows", "Install it from https://tectonic-typesetting.github.io, then rerun this command."},
	}
	for _, tc := range tests {
		t.Run(tc.goos, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			stub := runtest.NewStub(t)
			stub.Register("tectonic", func(run.Cmd) error { return fmt.Errorf("exec: %w", run.ErrNotFound) })

			err := tectonic.New(stub, ios, tc.goos).Build(context.Background(), "invoice.tex")

			var notInstalled *billing.ToolMissingError
			if !errors.As(err, &notInstalled) {
				t.Fatalf("Build error = %v, want *billing.ToolMissingError", err)
			}
			if got, want := err.Error(), "tectonic not found in PATH"; got != want {
				t.Fatalf("Error() = %q, want %q", got, want)
			}
			if got := notInstalled.Hint; got != tc.hint {
				t.Fatalf("Hint = %q, want %q", got, tc.hint)
			}
		})
	}
}
