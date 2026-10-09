package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestExitCodeMapsErrorTypes(t *testing.T) {
	root := &cobra.Command{Use: "invox"}
	customer := &cobra.Command{Use: "customer"}
	customerList := &cobra.Command{Use: "list"}
	customer.AddCommand(customerList)
	newCmd := &cobra.Command{Use: "new"}
	validate := &cobra.Command{Use: "validate"}
	root.AddCommand(customer, newCmd, validate)

	tests := []struct {
		name       string
		cmd        *cobra.Command
		err        error
		wantCode   int
		wantStderr string
	}{
		{"success", root, nil, 0, ""},
		{"help shown", root, flag.ErrHelp, 0, ""},
		{"help flag with a value", validate, &cmdutil.FlagError{Err: flag.ErrHelp}, 2, "error: flag: help requested\nRun 'invox validate --help' for usage.\n"},
		{"runtime error", newCmd, errors.New("disk full"), 1, "error: disk full\n"},
		{"usage error", customerList, cmdutil.FlagErrorf("unexpected arguments: x"), 2, "error: unexpected arguments: x\nRun 'invox customer list --help' for usage.\n"},
		{"root usage error", root, cmdutil.FlagErrorf("missing subcommand"), 2, "error: missing subcommand\nRun 'invox --help' for usage.\n"},
		{"global flag error", customerList, &cmdutil.FlagError{Err: errors.New("flag needs an argument: --config"), Root: true}, 2, "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n"},
		{"wrapped usage error", newCmd, fmt.Errorf("parse: %w", cmdutil.FlagErrorf("bad")), 2, "error: bad\nRun 'invox new --help' for usage.\n"},
		{"already reported", root, cmdutil.SilentError, 1, ""},
		{"cancelled", root, cmdutil.CancelError, 2, ""},
		{"external program", root, &billing.ToolFailedError{Tool: "tectonic", Code: 3, Err: errors.New("exit status 3")}, 1, "error: tectonic exited with status 3\n"},
		{"external program killed", root, &billing.ToolFailedError{Tool: "tectonic", Code: -1, Err: errors.New("signal: killed")}, 1, "error: tectonic failed: signal: killed\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, stderr := iostreams.Test()
			if got := exitCode(errorFactory(ios, t.TempDir()), tt.cmd, tt.err); got != tt.wantCode {
				t.Errorf("exitCode = %d, want %d", got, tt.wantCode)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}

// errorFactory is a Factory whose stderr is ios's and whose working
// directory is cwd.
func errorFactory(ios *iostreams.IOStreams, cwd string) *cmdutil.Factory {
	return &cmdutil.Factory{IOStreams: ios, Env: env.Env{Getwd: func() (string, error) { return cwd, nil }}}
}

// The working directory is the one invox was given, not the process's.
func TestRuntimeErrorShowsPathsRelativeToWorkingDir(t *testing.T) {
	cwd := t.TempDir()
	outside := filepath.Join(t.TempDir(), "b.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(errorFactory(ios, cwd), nil, fmt.Errorf("%s and %s", filepath.Join(cwd, "sub", "a.yaml"), outside))

	if want := "error: " + filepath.Join("sub", "a.yaml") + " and " + outside + "\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestMissingTectonicPrintsInstallHint(t *testing.T) {
	ios, _, _, stderr := iostreams.Test()

	exitCode(errorFactory(ios, t.TempDir()), nil, fmt.Errorf("build: %w", &billing.ToolMissingError{Tool: "tectonic", Hint: "Install it with 'brew install tectonic', then rerun this command."}))

	if want := "error: build: tectonic not found in PATH\nInstall it with 'brew install tectonic', then rerun this command.\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRuntimeErrorKeepsPathsThatOnlyContainWorkingDir(t *testing.T) {
	cwd := t.TempDir()
	mirror := filepath.Join(t.TempDir(), "mirror") + filepath.Join(cwd, "bad.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(errorFactory(ios, cwd), nil, fmt.Errorf("%s: invalid", mirror))

	if want := "error: " + mirror + ": invalid\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}
