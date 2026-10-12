package cli

import (
	"slices"
	"testing"
)

func TestVersionFlagToCommand(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"--version"}, want: []string{"version"}},
		{args: []string{"--version", "extra"}, want: []string{"version", "extra"}},
		{args: []string{"--no-input", "--config", "c.yaml", "--config=d.yaml", "--version"}, want: []string{"--no-input", "--config", "c.yaml", "--config=d.yaml", "version"}},
		{args: []string{"new", "--version"}, want: []string{"new", "--version"}},
		{args: []string{}, want: []string{}},
	}
	for _, tc := range tests {
		if got := versionFlagToCommand(tc.args); !slices.Equal(got, tc.want) {
			t.Errorf("versionFlagToCommand(%q) = %q, want %q", tc.args, got, tc.want)
		}
	}
}
