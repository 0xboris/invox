package cli_test

import (
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestCustomerAndConfigHelpMatchesHelpCommand(t *testing.T) {
	x := clitest.New(t)

	tests := []struct {
		topic    []string
		requests [][]string
	}{
		{topic: []string{"help", "customer"}, requests: [][]string{{"customer"}, {"customer", "-h"}, {"customer", "--help"}}},
		{topic: []string{"help", "customer", "list"}, requests: [][]string{{"customer", "list", "-h"}, {"customer", "list", "-c", "x.yaml", "--help"}}},
		{topic: []string{"help", "customer", "edit"}, requests: [][]string{{"customer", "edit", "-h"}}},
		{topic: []string{"help", "config"}, requests: [][]string{{"config", "-h"}, {"config", "--help"}}},
		{topic: []string{"help", "config", "edit"}, requests: [][]string{{"config", "edit", "-h"}}},
		{topic: []string{"help", "config", "paths"}, requests: [][]string{{"config", "paths", "-h"}}},
	}
	for _, tc := range tests {
		exitCode, want, stderr := x.Run(tc.topic)
		if exitCode != 0 || stderr != "" || want == "" {
			t.Fatalf("%q: exit code %d, stdout %q, stderr %q", tc.topic, exitCode, want, stderr)
		}
		for _, args := range tc.requests {
			exitCode, stdout, stderr := x.Run(args)
			if exitCode != 0 || stderr != "" || stdout != want {
				t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the %q page", args, exitCode, stderr, stdout, tc.topic)
			}
		}
	}
}
