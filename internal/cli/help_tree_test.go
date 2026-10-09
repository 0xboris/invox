package cli_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/clitest"
)

// TestEveryCommandHelpHasAnExample walks the command tree: every command's
// help renders on stdout, matches `invox help CMD`, and ends in examples
// with `$ invox` prompts. Every Short is one line without a period.
func TestEveryCommandHelpHasAnExample(t *testing.T) {
	var walk func(cmd *cobra.Command, path []string)
	walk = func(cmd *cobra.Command, path []string) {
		t.Run("invox "+strings.Join(path, " "), func(t *testing.T) {
			x := clitest.New(t)

			if cmd.Short == "" || strings.HasSuffix(cmd.Short, ".") || strings.Contains(cmd.Short, "\n") {
				t.Errorf("Short = %q, want one line without a trailing period", cmd.Short)
			}
			exitCode, stdout, stderr := x.Run(append(append([]string{}, path...), "--help"))
			if exitCode != 0 || stderr != "" {
				t.Fatalf("--help: exit %d, stderr %q", exitCode, stderr)
			}
			if !strings.Contains(stdout, "\nExamples:\n  $ invox ") {
				t.Errorf("help has no `$ invox` example:\n%s", stdout)
			}
			if !strings.Contains(stdout, "\nUsage:\n  invox") {
				t.Errorf("help has no usage:\n%s", stdout)
			}
			_, viaHelp, _ := x.Run(append([]string{"help"}, path...))
			if viaHelp != stdout {
				t.Errorf("`invox help %s` differs from --help:\n%s", strings.Join(path, " "), viaHelp)
			}
		})
		for _, sub := range cmd.Commands() {
			if sub.IsAvailableCommand() {
				walk(sub, append(append([]string{}, path...), sub.Name()))
			}
		}
	}
	walk(cli.NewRootCmd(clitest.New(t).Factory()), nil)
}
