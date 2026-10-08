package cli

import (
	"slices"
	"testing"
)

func TestSplitGlobalFlagsRemovesNoInput(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		want  []string
		found bool
	}{
		{name: "absent", args: []string{"config"}, want: []string{"config"}},
		{name: "before the command", args: []string{"--no-input", "config"}, want: []string{"config"}, found: true},
		{name: "after the command", args: []string{"new", "CUST-001", "-e", "--no-input"}, want: []string{"new", "CUST-001", "-e"}, found: true},
		{name: "twice", args: []string{"--no-input", "config", "--no-input"}, want: []string{"config"}, found: true},
		{name: "after --", args: []string{"new", "--", "--no-input"}, want: []string{"new", "--", "--no-input"}},
		{name: "before and after --", args: []string{"--no-input", "new", "--", "--no-input"}, want: []string{"new", "--", "--no-input"}, found: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, got, err := splitGlobalFlags(tc.args)
			if err != nil || !slices.Equal(got, tc.want) || g.noInput != tc.found {
				t.Fatalf("splitGlobalFlags(%q) = (%+v, %q, %v), want noInput %v and %q", tc.args, g, got, err, tc.found, tc.want)
			}
		})
	}
}
