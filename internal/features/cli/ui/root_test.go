package cli

import (
	"bytes"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// testCommands returns `template list --names` and `version`, the shape of
// the invox commands that the flag rewriting tests use.
func testCommands() []*cobra.Command {
	noop := func(*cobra.Command, []string) error { return nil }
	list := &cobra.Command{Use: "list", RunE: noop}
	list.Flags().Bool("names", false, "Print only the names")
	template := &cobra.Command{Use: "template"}
	template.AddCommand(list)
	version := &cobra.Command{Use: "version", RunE: noop}
	return []*cobra.Command{template, version}
}

func TestNormalizeLongFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     []string
		wantWarn string
	}{
		{
			name:     "single-dash long flag",
			args:     []string{"template", "list", "-names"},
			want:     []string{"template", "list", "--names"},
			wantWarn: "warning: -names is deprecated; use --names\n",
		},
		{
			name:     "single-dash long flag with value",
			args:     []string{"template", "list", "-names=false"},
			want:     []string{"template", "list", "--names=false"},
			wantWarn: "warning: -names is deprecated; use --names\n",
		},
		{
			name: "global flags",
			args: []string{"-no-input", "template", "list", "-config", "-names"},
			want: []string{"--no-input", "template", "list", "--config", "-names"},
			wantWarn: "warning: -no-input is deprecated; use --no-input\n" +
				"warning: -config is deprecated; use --config\n",
		},
		{
			name:     "help flag",
			args:     []string{"version", "-help"},
			want:     []string{"version", "--help"},
			wantWarn: "warning: -help is deprecated; use --help\n",
		},
		{
			name: "double-dash and shorthand flags",
			args: []string{"template", "list", "--names", "-h"},
			want: []string{"template", "list", "--names", "-h"},
		},
		{
			name: "after --",
			args: []string{"template", "list", "--", "-names"},
			want: []string{"template", "list", "--", "-names"},
		},
		{
			name: "unknown single-dash flag",
			args: []string{"template", "list", "-bogus"},
			want: []string{"template", "list", "--bogus"},
		},
		{
			name: "shorthand cluster and negative number",
			args: []string{"template", "list", "-hx", "-5"},
			want: []string{"template", "list", "-hx", "-5"},
		},
		{
			name: "unknown word on the help command",
			args: []string{"help", "-names"},
			want: []string{"help", "--names"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			var warn bytes.Buffer
			root, _ := newRootCmd(&Factory{IOStreams: ios}, testCommands())
			got := normalizeLongFlags(root, tc.args, &warn)
			if !slices.Equal(got, tc.want) {
				t.Errorf("args = %q, want %q", got, tc.want)
			}
			if warn.String() != tc.wantWarn {
				t.Errorf("warnings = %q, want %q", warn.String(), tc.wantWarn)
			}
		})
	}
}

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
