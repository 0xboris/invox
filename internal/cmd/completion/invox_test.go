package completion_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestCompletionPrintsAScriptForEachShell(t *testing.T) {
	for shell, want := range map[string]string{
		"bash":       "complete -o default -F __start_invox invox",
		"zsh":        "compdef _invox invox",
		"fish":       "complete -c invox ",
		"powershell": "Register-ArgumentCompleter -CommandName 'invox'",
	} {
		t.Run(shell, func(t *testing.T) {
			x := clitest.New(t)

			exitCode, stdout, stderr := x.Run([]string{"completion", shell})
			if exitCode != 0 || stderr != "" {
				t.Fatalf("exitCode = %d, stderr = %q, want 0 and empty", exitCode, stderr)
			}
			if !strings.Contains(stdout, want) || !strings.Contains(stdout, "__complete") {
				t.Fatalf("stdout does not contain %q and __complete:\n%s", want, stdout)
			}
		})
	}
}
