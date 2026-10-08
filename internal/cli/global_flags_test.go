package cli

import (
	"strings"
	"testing"
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exitCode, stdout, stderr := captureRun(t, tc.args)
			if exitCode != 2 || stdout != "" || !strings.HasPrefix(stderr, tc.wantPrefix) {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit 2, no stdout, stderr starting %q", exitCode, stdout, stderr, tc.wantPrefix)
			}
		})
	}
}
