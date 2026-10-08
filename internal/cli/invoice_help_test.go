package cli

import "testing"

func TestInvoiceCommandHelpMatchesHelpCommand(t *testing.T) {
	withFlag := map[string][]string{
		"new":       {"new", "CUST-001", "-e", "--help"},
		"increment": {"increment", "-i", "x.yaml", "--help"},
		"validate":  {"validate", "-i", "x.yaml", "--help"},
		"render":    {"render", "-i", "x.yaml", "--help"},
	}
	for _, name := range []string{"new", "increment", "validate", "render"} {
		exitCode, want, stderr := captureRun(t, []string{"help", name})
		if exitCode != 0 || stderr != "" || want == "" {
			t.Fatalf("help %s: exit code %d, stdout %q, stderr %q", name, exitCode, want, stderr)
		}
		for _, args := range [][]string{{name, "-h"}, {name, "--help"}, withFlag[name]} {
			exitCode, stdout, stderr := captureRun(t, args)
			if exitCode != 0 || stderr != "" || stdout != want {
				t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the %s page", args, exitCode, stderr, stdout, name)
			}
		}
	}
}
