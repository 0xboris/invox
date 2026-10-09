package cli_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

// --no-input works anywhere before --, and is an argument after it.
func TestNoInputAnywhereBeforeDoubleDash(t *testing.T) {
	const disabled = "error: cannot open an editor: prompts are disabled (--no-input or INVOX_PROMPT_DISABLED)"
	tests := []struct {
		name       string
		args       []string
		wantPrefix string
	}{
		{name: "absent", args: []string{"config"}, wantPrefix: "error: cannot open an editor: stdin is not a terminal"},
		{name: "before the command", args: []string{"--no-input", "config"}, wantPrefix: disabled},
		{name: "after the command", args: []string{"config", "--no-input"}, wantPrefix: disabled},
		{name: "twice", args: []string{"--no-input", "config", "--no-input"}, wantPrefix: disabled},
		{name: "after --", args: []string{"config", "--", "--no-input"}, wantPrefix: "error: unexpected arguments: --no-input"},
		{name: "before and after --", args: []string{"--no-input", "config", "--", "--no-input"}, wantPrefix: "error: unexpected arguments: --no-input"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			exitCode, stdout, stderr := x.Run(tc.args)
			if exitCode != 2 || stdout != "" || !strings.HasPrefix(stderr, tc.wantPrefix) {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit 2, no stdout, stderr starting %q", exitCode, stdout, stderr, tc.wantPrefix)
			}
		})
	}
}

// --no-input after the command's own flags still keeps new -e from opening
// an editor, after it created the invoice.
func TestNoInputAfterNewEdit(t *testing.T) {
	x := clitest.New(t)

	if exitCode, _, stderr := x.Run([]string{"init"}); exitCode != 0 {
		t.Fatalf("init: exit %d, stderr %q", exitCode, stderr)
	}
	x.Chdir(t.TempDir())

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-e", "--no-input"})
	const want = "but cannot open an editor: prompts are disabled (--no-input or INVOX_PROMPT_DISABLED)"
	if exitCode != 2 || stdout != "" || !strings.HasPrefix(stderr, "error: created ") || !strings.Contains(stderr, want) {
		t.Fatalf("exit %d, stdout %q, stderr %q; want exit 2, no stdout, an error that the invoice was created %s", exitCode, stdout, stderr, want)
	}
}
