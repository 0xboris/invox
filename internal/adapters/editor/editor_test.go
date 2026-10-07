package editor_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestEditRunsTheEditorThroughTheShell(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		want run.Cmd
	}{
		{
			name: "VISUAL wins over EDITOR",
			goos: "linux",
			env:  map[string]string{"VISUAL": "code -w", "EDITOR": "vim", "SHELL": "/bin/zsh"},
			want: run.Cmd{
				Name: "/bin/zsh",
				Args: []string{"-lc", `eval "$INVOX_EDITOR" '"$1"'`, "invox", "invoice.yaml"},
				Env:  []string{"INVOX_EDITOR=code -w"},
			},
		},
		{
			name: "EDITOR without SHELL",
			goos: "darwin",
			env:  map[string]string{"EDITOR": " nano "},
			want: run.Cmd{
				Name: "/bin/sh",
				Args: []string{"-lc", `eval "$INVOX_EDITOR" '"$1"'`, "invox", "invoice.yaml"},
				Env:  []string{"INVOX_EDITOR=nano"},
			},
		},
		{
			name: "vi by default",
			goos: "linux",
			env:  map[string]string{},
			want: run.Cmd{
				Name: "/bin/sh",
				Args: []string{"-lc", `eval "$INVOX_EDITOR" '"$1"'`, "invox", "invoice.yaml"},
				Env:  []string{"INVOX_EDITOR=vi"},
			},
		},
		{
			name: "Windows runs cmd",
			goos: "windows",
			env:  map[string]string{"EDITOR": "code -w"},
			want: run.Cmd{
				Name: "cmd",
				Args: []string{"/c", "code -w", "invoice.yaml"},
				Env:  []string{"INVOX_EDITOR=code -w"},
			},
		},
		{
			name: "notepad by default on Windows",
			goos: "windows",
			env:  map[string]string{},
			want: run.Cmd{
				Name: "cmd",
				Args: []string{"/c", "notepad", "invoice.yaml"},
				Env:  []string{"INVOX_EDITOR=notepad"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			stub := run.NewStub(t)
			var got run.Cmd
			stub.Register(tc.want.Name, func(cmd run.Cmd) error {
				got = cmd
				return nil
			})
			getenv := func(key string) string { return tc.env[key] }

			if err := editor.New(stub, ios, tc.goos, getenv).Edit(context.Background(), "invoice.yaml"); err != nil {
				t.Fatalf("Edit returned error: %v", err)
			}

			if !slices.Equal(got.Args, tc.want.Args) {
				t.Fatalf("Args = %q, want %q", got.Args, tc.want.Args)
			}
			if !slices.Equal(got.Env, tc.want.Env) {
				t.Fatalf("Env = %q, want %q", got.Env, tc.want.Env)
			}
		})
	}
}

func TestEditConnectsStdinAndSendsOutputToStderr(t *testing.T) {
	ios, stdin, stdout, stderr := iostreams.Test()
	stdin.WriteString(":wq\n")
	stub := run.NewStub(t)
	var got run.Cmd
	var input []byte
	stub.Register("/bin/sh", func(cmd run.Cmd) error {
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
	stub := run.NewStub(t)
	stub.Register("/bin/sh", func(run.Cmd) error { return &run.ExecError{Name: "/bin/sh", Code: 1} })

	err := editor.New(stub, ios, "linux", func(string) string { return "" }).Edit(context.Background(), "a.yaml")

	if err == nil || err.Error() != "exit status 1" {
		t.Fatalf("Edit error = %v, want exit status 1", err)
	}
}
