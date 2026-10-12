package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// A single-dash word that names a long flag, or that starts with no
// shorthand, is a usage error. Nothing is rewritten, so -output=x.pdf can't
// become -o utput=x.pdf.
func TestSingleDashLongFlagsFail(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "long flag",
			args:       []string{"template", "list", "-names"},
			wantStderr: "error: -names is not a flag; use --names\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "long flag with value",
			args:       []string{"template", "list", "-names=false"},
			wantStderr: "error: -names is not a flag; use --names\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "long flag that has a shorthand",
			args:       []string{"new", "CUST-001", "-output=x.pdf"},
			wantStderr: "error: -output is not a flag; use --output\nRun 'invox new --help' for usage.\n",
		},
		{
			name:       "global flag",
			args:       []string{"-no-input", "template", "list"},
			wantStderr: "error: -no-input is not a flag; use --no-input\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "help flag",
			args:       []string{"version", "-help"},
			wantStderr: "error: -help is not a flag; use --help\nRun 'invox version --help' for usage.\n",
		},
		{
			name:       "version flag on the root",
			args:       []string{"-version"},
			wantStderr: "error: -version is not a flag; use --version\nRun 'invox --help' for usage.\n",
		},
		{
			name:       "unknown word",
			args:       []string{"template", "list", "-bogus"},
			wantStderr: "error: unknown flag: -bogus\nRun 'invox template list --help' for usage.\n",
		},
		{
			name:       "after an unknown subcommand",
			args:       []string{"send", "-input", "x.pdf", "--to", "a@example.com"},
			wantStderr: "error: unknown subcommand \"send\"; did you mean \"email\"?\nRun 'invox --help' for usage.\n",
		},
		{
			name:       "after --",
			args:       []string{"template", "list", "--", "-names"},
			wantStderr: "error: unexpected arguments: -names\nRun 'invox template list --help' for usage.\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			dir := t.TempDir()
			x.Chdir(dir)
			exitCode, stdout, stderr := x.Run(tc.args)
			if exitCode != 2 {
				t.Errorf("exit code = %d, want 2", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tc.wantStderr)
			}
			if entries, _ := os.ReadDir(dir); len(entries) > 0 {
				t.Errorf("the working directory has %s, want nothing written", entries[0].Name())
			}
		})
	}
}

// pflag alone reads -output=x.yaml as -o utput=x.yaml, so new would write
// utput=x.yaml.
func TestSingleDashOutputWritesNothing(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-output=x.yaml", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if want := "error: -output is not a flag; use --output\nRun 'invox new --help' for usage.\n"; exitCode != 2 || stdout != "" || stderr != want {
		t.Fatalf("new -output=x.yaml = (%d, %q, %q), want (2, \"\", %q)", exitCode, stdout, stderr, want)
	}
	if entries, _ := os.ReadDir(workDir); len(entries) > 0 {
		t.Fatalf("new -output=x.yaml wrote %s, want nothing written", entries[0].Name())
	}
}

// A shorthand with its value attached, and a flag value that starts with a
// dash, are not single-dash long flags.
func TestShorthandGroupsStillWork(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-ofile.yaml", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if want := "Created file.yaml for CUST-001 (CUST-001-002)\n"; exitCode != 0 || stdout != "file.yaml\n" || stderr != want {
		t.Fatalf("new -ofile.yaml = (%d, %q, %q), want (0, %q, %q)", exitCode, stdout, stderr, "file.yaml\n", want)
	}
	if _, err := os.Stat(filepath.Join(workDir, "file.yaml")); err != nil {
		t.Fatalf("new -ofile.yaml did not write file.yaml: %v", err)
	}

	exitCode, stdout, stderr = x.Run([]string{"new", "CUST-001", "-n", "-o", "-names.yaml", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if want := "Would create -names.yaml for CUST-001 (CUST-001-003)\n"; exitCode != 0 || stdout != "-names.yaml\n" || stderr != want {
		t.Fatalf("new -o -names.yaml = (%d, %q, %q), want (0, %q, %q)", exitCode, stdout, stderr, "-names.yaml\n", want)
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
			wantStderr: "error: unknown flag: -nmaes; did you mean --names?\nRun 'invox template list --help' for usage.\n",
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
			wantStderr: "error: unexpected arguments: extra\nRun 'invox version --help' for usage.\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			exitCode, stdout, stderr := x.Run(tc.args)
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
	x := clitest.New(t)

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
		exitCode, want, stderr := x.Run(tc.topic)
		if exitCode != 0 || stderr != "" {
			t.Fatalf("%q: exit code %d, stderr %q", tc.topic, exitCode, stderr)
		}
		if first, _, _ := bytes.Cut([]byte(want), []byte("\n")); string(first)+"\n" != tc.wantFirst {
			t.Errorf("%q: first line = %q, want %q", tc.topic, first, tc.wantFirst)
		}
		for _, args := range tc.requests {
			exitCode, stdout, stderr := x.Run(args)
			if exitCode != 0 || stderr != "" || stdout != want {
				t.Errorf("%q: exit code %d, stderr %q, stdout %q; want 0, empty, the %q page", args, exitCode, stderr, stdout, tc.topic)
			}
		}
	}
}

func TestCobraCommandErrorBecomesSignal(t *testing.T) {
	for _, tc := range signalCases {
		t.Run(tc.name, func(t *testing.T) {
			f := clitest.New(t).Factory()
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(&cli.SignalError{Signal: tc.signal})

			exitCode := cli.MainContext(ctx, []string{"template", "list", "--bogus"}, f)
			if exitCode != tc.exitCode {
				t.Errorf("exit code = %d, want %d", exitCode, tc.exitCode)
			}
		})
	}
}

func TestCobraCommandAppliesGlobalFlags(t *testing.T) {
	f := clitest.New(t).Factory()
	exitCode := cli.MainContext(context.Background(), []string{"version", "--no-input", "--config", "custom.yaml"}, f)
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
