package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// getwdNone is a getwd that fails, so exitCode leaves paths as they are.
func getwdNone() (string, error) { return "", errors.New("no working directory") }

// getwdAt is a getwd that returns dir.
func getwdAt(dir string) func() (string, error) {
	return func() (string, error) { return dir, nil }
}

func TestExitCodeMapsErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStderr string
	}{
		{"success", nil, 0, ""},
		{"help shown", flag.ErrHelp, 0, ""},
		{"help flag with a value", &cmdutil.FlagError{Command: "validate", Err: flag.ErrHelp}, 2, "error: flag: help requested\nRun 'invox validate --help' for usage.\n"},
		{"runtime error", errors.New("disk full"), 1, "error: disk full\n"},
		{"usage error", cmdutil.FlagErrorf("customer list", "unexpected arguments: x"), 2, "error: unexpected arguments: x\nRun 'invox customer list --help' for usage.\n"},
		{"root usage error", cmdutil.FlagErrorf("", "missing subcommand"), 2, "error: missing subcommand\nRun 'invox --help' for usage.\n"},
		{"wrapped usage error", fmt.Errorf("parse: %w", cmdutil.FlagErrorf("new", "bad")), 2, "error: bad\nRun 'invox new --help' for usage.\n"},
		{"already reported", cmdutil.SilentError, 1, ""},
		{"cancelled", cmdutil.CancelError, 2, ""},
		{"external program", &cmdutil.ExecError{Program: "tectonic", Code: 3, Err: errors.New("exit status 3")}, 1, "error: tectonic exited with status 3\n"},
		{"external program killed", &cmdutil.ExecError{Program: "tectonic", Code: -1, Err: errors.New("signal: killed")}, 1, "error: tectonic failed: signal: killed\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, stderr := iostreams.Test()
			if got := exitCode(ios, getwdNone, tt.err); got != tt.wantCode {
				t.Errorf("exitCode = %d, want %d", got, tt.wantCode)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}

func TestRuntimeErrorShowsPathsRelativeToWorkingDir(t *testing.T) {
	cwd := t.TempDir()
	outside := filepath.Join(t.TempDir(), "b.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, getwdAt(cwd), fmt.Errorf("%s and %s", filepath.Join(cwd, "sub", "a.yaml"), outside))

	if want := "error: " + filepath.Join("sub", "a.yaml") + " and " + outside + "\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestMissingTectonicPrintsInstallHint(t *testing.T) {
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, getwdNone, fmt.Errorf("build: %w", &ports.CompilerNotInstalledError{GOOS: "darwin"}))

	if want := "error: build: tectonic not found in PATH\nInstall it with 'brew install tectonic', then rerun this command.\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRuntimeErrorKeepsPathsThatOnlyContainWorkingDir(t *testing.T) {
	cwd := t.TempDir()
	mirror := filepath.Join(t.TempDir(), "mirror") + filepath.Join(cwd, "bad.yaml")
	ios, _, _, stderr := iostreams.Test()

	exitCode(ios, getwdAt(cwd), fmt.Errorf("%s: invalid", mirror))

	if want := "error: " + mirror + ": invalid\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}
