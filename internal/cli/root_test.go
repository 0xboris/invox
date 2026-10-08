package cli

import (
	"bytes"
	"context"
	"slices"
	"testing"
)

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
			name: "legacy command",
			args: []string{"build", "x.yaml", "-archive"},
			want: []string{"build", "x.yaml", "-archive"},
		},
		{
			name: "help command",
			args: []string{"help", "-names"},
			want: []string{"help", "-names"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := testFactory(t)
			var warn bytes.Buffer
			got := normalizeLongFlags(newRootCmd(f), tc.args, &warn)
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

func TestCobraCommandUsageErrors(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "unknown flag",
			args:       []string{"template", "list", "--bogus"},
			wantStderr: "error: unknown flag: --bogus\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "unknown single-dash long flag",
			args:       []string{"template", "list", "-nmaes"},
			wantStderr: "error: unknown flag: --nmaes; did you mean --names?\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "invalid boolean",
			args:       []string{"template", "list", "--names=bogus"},
			wantStderr: "error: invalid value \"bogus\" for --names\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "unknown shorthand",
			args:       []string{"version", "-x"},
			wantStderr: "error: unknown shorthand flag: -x\nRun 'invox version --help' for usage.\n",
		},
		{
			name:       "empty config",
			args:       []string{"template", "list", "--config="},
			wantStderr: "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n",
		},
		{
			name:       "global flag without value",
			args:       []string{"template", "list", "--config"},
			wantStderr: "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n",
		},
		{
			name:       "extra argument",
			args:       []string{"version", "extra"},
			wantStderr: "error: unexpected arguments for version: extra\nRun 'invox version --help' for usage.\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exitCode, stdout, stderr := captureRun(t, tc.args)
			if exitCode != 2 {
				t.Errorf("exit code = %d, want 2", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tc.wantStderr)
			}
		})
	}
}

func TestCobraCommandHelpMatchesHelpCommand(t *testing.T) {
	tests := []struct {
		topic     []string
		requests  [][]string
		wantFirst string
	}{
		{
			topic:     []string{"help", "version"},
			requests:  [][]string{{"version", "--help"}, {"version", "-h"}},
			wantFirst: "Show the invox version.\n",
		},
		{
			topic:     []string{"help", "template"},
			requests:  [][]string{{"template"}, {"template", "--help"}, {"template", "-h"}, {"--no-input", "template", "-h"}},
			wantFirst: "Template-related commands.\n",
		},
		{
			topic:     []string{"help", "template", "list"},
			requests:  [][]string{{"template", "list", "--help"}, {"template", "list", "-h"}, {"template", "list", "--names", "-h"}},
			wantFirst: "List available LaTeX invoice templates from the same directory as the resolved default template.\n",
		},
	}
	for _, tc := range tests {
		exitCode, want, stderr := captureRun(t, tc.topic)
		if exitCode != 0 || stderr != "" {
			t.Fatalf("%q: exit code %d, stderr %q", tc.topic, exitCode, stderr)
		}
		if first, _, _ := bytes.Cut([]byte(want), []byte("\n")); string(first)+"\n" != tc.wantFirst {
			t.Errorf("%q: first line = %q, want %q", tc.topic, first, tc.wantFirst)
		}
		for _, args := range tc.requests {
			exitCode, stdout, stderr := captureRun(t, args)
			if exitCode != 0 || stderr != "" || stdout != want {
				t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the %q page", args, exitCode, stderr, stdout, tc.topic)
			}
		}
	}
}

func TestCobraCommandErrorBecomesSignal(t *testing.T) {
	for _, tc := range signalCases {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := testFactory(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(&SignalError{Signal: tc.signal})

			exitCode := mainContext(ctx, []string{"template", "list", "--bogus"}, f)
			if exitCode != tc.exitCode {
				t.Errorf("exit code = %d, want %d", exitCode, tc.exitCode)
			}
		})
	}
}

func TestCobraCommandAppliesGlobalFlags(t *testing.T) {
	f, _ := testFactory(t)
	exitCode := mainContext(context.Background(), []string{"version", "--no-input", "--config", "custom.yaml"}, f)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if !f.IOStreams.PromptDisabled() {
		t.Error("--no-input after the command did not disable prompts")
	}
	if f.ConfigFile != "custom.yaml" {
		t.Errorf("ConfigFile = %q, want %q", f.ConfigFile, "custom.yaml")
	}
}
