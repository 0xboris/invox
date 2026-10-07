package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	t.Setenv("SHELL", "/bin/sh")
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
