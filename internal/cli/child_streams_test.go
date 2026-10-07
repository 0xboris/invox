package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/iostreams"
)

// fakeChildEnv makes the test binary print the base names of its arguments
// to stdout and exit, standing in for an editor or an opener.
const fakeChildEnv = "INVOX_TEST_FAKE_CHILD"

func runFakeChild(args []string) int {
	for _, arg := range args {
		fmt.Printf("fake child: %s\n", filepath.Base(arg))
	}
	return 0
}

// fakeShellEnv makes the test binary stand in for the login shell invox runs
// the editor with, so no login profile can print into the editor's output.
const fakeShellEnv = "INVOX_TEST_FAKE_SHELL"

// runFakeShell stands in for the shell invox runs the editor with:
//
//	$SHELL -lc 'eval "$INVOX_EDITOR" '"$1"'' invox FILE
//
// It runs $INVOX_EDITOR, split on spaces, with FILE appended and this
// process's standard streams, and drops fakeShellEnv from the editor's
// environment so the editor runs as a fake child.
func runFakeShell(args []string) int {
	if len(args) != 4 || args[0] != "-lc" || args[2] != "invox" {
		fmt.Fprintf(os.Stderr, "fake shell: unexpected arguments %q\n", args)
		return 2
	}
	editor := strings.Fields(os.Getenv("INVOX_EDITOR"))
	if len(editor) == 0 {
		fmt.Fprintln(os.Stderr, "fake shell: INVOX_EDITOR is empty")
		return 2
	}

	cmd := exec.Command(editor[0], append(editor[1:], args[3])...)
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, fakeShellEnv+"=")
	})
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "fake shell: %v\n", err)
		return 1
	}
	return 0
}

func testExecutable(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned error: %v", err)
	}
	return self
}

func TestRunChildSendsChildStdoutToStderr(t *testing.T) {
	t.Setenv(fakeChildEnv, "1")
	ios, _, stdout, stderr := iostreams.Test()

	if err := runChild(ios, exec.Command(testExecutable(t), "draft.eml")); err != nil {
		t.Fatalf("runChild returned error: %v", err)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got, want := stderr.String(), "fake child: draft.eml\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestOpenTextFileSendsEditorStdoutToStderr(t *testing.T) {
	t.Setenv(fakeChildEnv, "1")
	t.Setenv("VISUAL", testExecutable(t))
	if runtime.GOOS != "windows" {
		t.Setenv(fakeShellEnv, "1")
		t.Setenv("SHELL", testExecutable(t))
	}
	ios, _, stdout, stderr := iostreams.Test()

	if err := defaultOpenTextFile(ios, filepath.Join(t.TempDir(), "invoice.yaml")); err != nil {
		t.Fatalf("defaultOpenTextFile returned error: %v", err)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got, want := stderr.String(), "fake child: invoice.yaml\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
