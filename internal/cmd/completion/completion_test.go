package completion

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdCompletionParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantShell string
		wantErr   string
	}{
		{name: "bash", args: []string{"bash"}, wantShell: "bash"},
		{name: "powershell", args: []string{"powershell"}, wantShell: "powershell"},
		{name: "no shell", args: []string{}},
		{name: "unsupported shell", args: []string{"tcsh"}, wantErr: `unsupported shell "tcsh"`},
		{name: "extra arguments", args: []string{"bash", "zsh"}, wantErr: "unexpected arguments: bash zsh"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := factorytest.New(t, ios, factorytest.Options{})
			var got *CompletionOptions
			cmd := NewCmdCompletion(f, func(opts *CompletionOptions) error {
				got = opts
				return nil
			})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()

			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
					t.Fatalf("Execute error = %#v, want FlagError %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if tc.wantShell == "" {
				if got != nil {
					t.Fatalf("runF called with %+v, want help only", got)
				}
				return
			}
			if got == nil || got.Shell != tc.wantShell || got.Root != cmd || got.IO != ios {
				t.Fatalf("runF opts = %+v, want Shell %q, Root the command tree and the factory's streams", got, tc.wantShell)
			}
		})
	}
}

func TestCompletionRunWritesTheScriptToStdout(t *testing.T) {
	ios, _, out, errOut := iostreams.Test()
	root := &cobra.Command{Use: "invox"}

	if err := completionRun(&CompletionOptions{IO: ios, Root: root, Shell: "zsh"}); err != nil {
		t.Fatalf("completionRun returned error: %v", err)
	}
	if !strings.HasPrefix(out.String(), "#compdef invox\n") {
		t.Errorf("stdout starts %q, want a zsh script for invox", out.String()[:min(out.Len(), 40)])
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr = %q, want empty", errOut.String())
	}
}
