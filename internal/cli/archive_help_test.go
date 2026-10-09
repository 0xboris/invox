package cli

import "testing"

func TestArchiveHelpMatchesHelpCommand(t *testing.T) {
	for _, tc := range []struct {
		topic    []string
		requests [][]string
	}{
		{topic: []string{"help", "archive"}, requests: [][]string{{"archive"}, {"archive", "-h"}, {"archive", "--help"}}},
		{topic: []string{"help", "archive", "add"}, requests: [][]string{{"archive", "add", "-h"}, {"archive", "add", "x.yaml", "--yes", "--help"}}},
		{topic: []string{"help", "archive", "edit"}, requests: [][]string{{"archive", "edit", "-h"}, {"archive", "edit", "x.yaml", "--help"}}},
		{topic: []string{"help", "archive", "list"}, requests: [][]string{{"archive", "list", "-h"}}},
	} {
		exitCode, want, stderr := captureRun(t, tc.topic)
		if exitCode != 0 || stderr != "" || want == "" {
			t.Fatalf("%q: exit code %d, stdout %q, stderr %q", tc.topic, exitCode, want, stderr)
		}
		for _, args := range tc.requests {
			exitCode, stdout, stderr := captureRun(t, args)
			if exitCode != 0 || stderr != "" || stdout != want {
				t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the %q page", args, exitCode, stderr, stdout, tc.topic)
			}
		}
	}
}
