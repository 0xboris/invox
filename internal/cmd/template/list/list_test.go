package list

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdListParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantNamesOnly bool
		wantErr       string
	}{
		{name: "no flags", args: []string{}},
		{name: "names", args: []string{"--names"}, wantNamesOnly: true},
		{name: "names false", args: []string{"--names=false"}},
		{name: "extra argument", args: []string{"extra"}, wantErr: "unexpected arguments: extra"},
		{name: "argument after --", args: []string{"--", "--names"}, wantErr: "unexpected arguments: --names"},
		{name: "misspelt flag", args: []string{"--nmaes"}, wantErr: "unknown flag: --nmaes; did you mean --names?"},
		{name: "unknown flag", args: []string{"--bogus"}, wantErr: "unknown flag: --bogus"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := cmdutil.NewFactory(ios, run.Exec{}, env.System())
			var got *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
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
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
					t.Fatalf("Execute error = %v, want FlagError %q", err, tc.wantErr)
				}
				if got != nil {
					t.Error("runF ran after a usage error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got == nil {
				t.Fatal("runF did not run")
			}
			if got.NamesOnly != tc.wantNamesOnly {
				t.Errorf("NamesOnly = %v, want %v", got.NamesOnly, tc.wantNamesOnly)
			}
			if got.IO != ios {
				t.Error("opts.IO is not the factory's streams")
			}
		})
	}
}

func TestListRun(t *testing.T) {
	configDir := t.TempDir()
	for _, name := range []string{"template.tex", "plain.tex", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f := factorytest.New(t, nil, factorytest.Options{ConfigDir: configDir})

	tests := []struct {
		name      string
		namesOnly bool
		want      string
	}{
		{
			name: "piped",
			want: "plain.tex\t" + tsvPath(configDir, "plain.tex") + "\n" +
				"template.tex\t" + tsvPath(configDir, "template.tex") + "\n",
		},
		{name: "names only", namesOnly: true, want: "plain.tex\ntemplate.tex\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, out, errOut := iostreams.Test()
			opts := &ListOptions{IO: ios, Service: f.Service, NamesOnly: tc.namesOnly}
			if err := listRun(opts); err != nil {
				t.Fatalf("listRun returned error: %v", err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("stdout = %q, want %q", got, tc.want)
			}
			if got := errOut.String(); got != "" {
				t.Errorf("stderr = %q, want empty", got)
			}
		})
	}
}

// tsvPath is the path to name in dir as piped output writes it, with each
// backslash of a Windows path escaped.
func tsvPath(dir, name string) string {
	return strings.ReplaceAll(filepath.Join(dir, name), `\`, `\\`)
}
