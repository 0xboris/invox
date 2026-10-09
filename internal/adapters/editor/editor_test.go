package editor_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestEditConnectsStdinAndSendsOutputToStderr(t *testing.T) {
	ios, stdin, stdout, stderr := iostreams.Test()
	stdin.WriteString(":wq\n")
	stub := runtest.NewStub(t)
	var got run.Cmd
	var input []byte
	stub.Register("vi", func(cmd run.Cmd) error {
		got = cmd
		input, _ = io.ReadAll(cmd.Stdin)
		fmt.Fprintln(cmd.Stdout, "screen")
		fmt.Fprintln(cmd.Stderr, "warning")
		return nil
	})

	if err := editor.New(stub, ios, "linux", func(string) string { return "" }).Edit(context.Background(), "a.yaml"); err != nil {
		t.Fatalf("Edit returned error: %v", err)
	}

	if got.Stdin != ios.In || got.Stdout != ios.ErrOut || got.Stderr != ios.ErrOut {
		t.Fatalf("streams = (%v, %v, %v), want ios.In, ios.ErrOut, ios.ErrOut themselves", got.Stdin, got.Stdout, got.Stderr)
	}
	if string(input) != ":wq\n" {
		t.Fatalf("stdin = %q, want %q", input, ":wq\n")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if want := "screen\nwarning\n"; stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestEditReturnsTheEditorsFailure(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	stub := runtest.NewStub(t)
	stub.Register("code", func(run.Cmd) error { return &run.ExecError{Name: "code", Code: 3} })
	getenv := func(key string) string { return map[string]string{"EDITOR": "code -w"}[key] }

	err := editor.New(stub, ios, "linux", getenv).Edit(context.Background(), "a.yaml")

	var execErr *run.ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("Edit error = %v, want *run.ExecError", err)
	}
	if execErr.Name != "code -w" || execErr.Code != 3 || err.Error() != "exit status 3" {
		t.Fatalf("Edit error = {Name: %q, Code: %d, %q}, want {Name: %q, Code: 3, %q}", execErr.Name, execErr.Code, err, "code -w", "exit status 3")
	}
}
