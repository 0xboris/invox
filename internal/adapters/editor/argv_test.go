package editor_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestEditRunsTheEditorSetting(t *testing.T) {
	const path = "/work/invoice.yaml"
	tests := []struct {
		name     string
		goos     string
		env      map[string]string
		wantName string
		wantArgs []string
	}{
		{
			name:     "EDITOR with a flag",
			goos:     "linux",
			env:      map[string]string{"EDITOR": "code -w"},
			wantName: "code",
			wantArgs: []string{"-w", path},
		},
		{
			name:     "EDITOR alone",
			goos:     "darwin",
			env:      map[string]string{"EDITOR": " vim "},
			wantName: "vim",
			wantArgs: []string{path},
		},
		{
			name:     "VISUAL wins over EDITOR",
			goos:     "linux",
			env:      map[string]string{"VISUAL": "nano", "EDITOR": "vim"},
			wantName: "nano",
			wantArgs: []string{path},
		},
		{
			name:     "quoted argument",
			goos:     "linux",
			env:      map[string]string{"EDITOR": `vim -c "set ft=yaml" -c 'set nu'`},
			wantName: "vim",
			wantArgs: []string{"-c", "set ft=yaml", "-c", "set nu", path},
		},
		{
			name:     "vi by default",
			goos:     "linux",
			env:      map[string]string{},
			wantName: "vi",
			wantArgs: []string{path},
		},
		{
			name:     "shell syntax runs through sh -c",
			goos:     "linux",
			env:      map[string]string{"EDITOR": "$HOME/bin/ed"},
			wantName: "/bin/sh",
			wantArgs: []string{"-c", `$HOME/bin/ed "$@"`, "sh", path},
		},
		{
			name:     "assignment prefix runs through sh -c",
			goos:     "darwin",
			env:      map[string]string{"VISUAL": "TERM=xterm vim"},
			wantName: "/bin/sh",
			wantArgs: []string{"-c", `TERM=xterm vim "$@"`, "sh", path},
		},
		{
			name:     "Windows path in double quotes",
			goos:     "windows",
			env:      map[string]string{"EDITOR": `"C:\Program Files\Ed\ed.exe" -w`},
			wantName: `C:\Program Files\Ed\ed.exe`,
			wantArgs: []string{"-w", path},
		},
		{
			name:     "Windows runs shell syntax directly",
			goos:     "windows",
			env:      map[string]string{"EDITOR": "ed.exe $x"},
			wantName: "ed.exe",
			wantArgs: []string{"$x", path},
		},
		{
			name:     "notepad by default on Windows",
			goos:     "windows",
			env:      map[string]string{},
			wantName: "notepad",
			wantArgs: []string{path},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			stub := runtest.NewStub(t)
			var got run.Cmd
			stub.Register(tc.wantName, func(cmd run.Cmd) error {
				got = cmd
				return nil
			})
			var asked []string
			getenv := func(key string) string {
				asked = append(asked, key)
				return tc.env[key]
			}

			if err := editor.New(stub, ios, tc.goos, getenv).Edit(context.Background(), path); err != nil {
				t.Fatalf("Edit returned error: %v", err)
			}

			if got.Name != tc.wantName || !slices.Equal(got.Args, tc.wantArgs) {
				t.Fatalf("ran %q %q, want %q %q", got.Name, got.Args, tc.wantName, tc.wantArgs)
			}
			for _, arg := range got.Args {
				if strings.HasPrefix(arg, "-l") {
					t.Fatalf("Args = %q, want no login-shell flag", got.Args)
				}
			}
			if len(got.Env) != 0 {
				t.Fatalf("Env = %q, want empty", got.Env)
			}
			for _, key := range asked {
				if key != "VISUAL" && key != "EDITOR" {
					t.Fatalf("Edit read %s, want only VISUAL and EDITOR", key)
				}
			}
		})
	}
}

func TestEditRejectsAnUnterminatedQuote(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	stub := runtest.NewStub(t)
	getenv := func(key string) string { return map[string]string{"EDITOR": "vim 'x"}[key] }

	err := editor.New(stub, ios, "linux", getenv).Edit(context.Background(), "a.yaml")

	if want := `cannot parse editor "vim 'x": unterminated quote`; err == nil || err.Error() != want {
		t.Fatalf("Edit error = %v, want %q", err, want)
	}
}
