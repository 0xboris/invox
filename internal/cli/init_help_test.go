package cli_test

import (
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestInitHelpMatchesHelpCommand(t *testing.T) {
	x := clitest.New(t)

	exitCode, want, stderr := x.Run([]string{"help", "init"})
	if exitCode != 0 || stderr != "" || want == "" {
		t.Fatalf("help init: exit code %d, stdout %q, stderr %q", exitCode, want, stderr)
	}
	for _, args := range [][]string{{"init", "-h"}, {"init", "--help"}} {
		exitCode, stdout, stderr := x.Run(args)
		if exitCode != 0 || stderr != "" || stdout != want {
			t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the init page", args, exitCode, stderr, stdout)
		}
	}
}
