package cli

import "testing"

func TestBuildAndEmailHelpMatchesHelpCommand(t *testing.T) {
	for _, tc := range []struct {
		topic    []string
		requests [][]string
	}{
		{topic: []string{"help", "build"}, requests: [][]string{{"build", "-h"}, {"build", "x.yaml", "--archive", "--help"}}},
		{topic: []string{"help", "email"}, requests: [][]string{{"email", "-h"}, {"email", "x.yaml", "--help"}}},
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
