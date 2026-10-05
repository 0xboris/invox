package iostreams

import (
	"io"
	"os"
	"testing"
)

func TestDevNullIsNotATerminal(t *testing.T) {
	t.Parallel()

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Open(%s) returned error: %v", os.DevNull, err)
	}
	t.Cleanup(func() { _ = devNull.Close() })

	ios := newSystem(devNull, devNull, devNull)
	if ios.IsStdinTTY() || ios.IsStderrTTY() {
		t.Fatalf("IsStdinTTY() = %v, IsStderrTTY() = %v on %s, want false", ios.IsStdinTTY(), ios.IsStderrTTY(), os.DevNull)
	}
	if ios.CanPrompt() {
		t.Fatalf("CanPrompt() = true on %s, want false", os.DevNull)
	}
}

func TestSystemForceTTY(t *testing.T) {
	t.Setenv("INVOX_FORCE_TTY", "1")

	if !System().IsStdoutTTY() {
		t.Fatal("IsStdoutTTY() = false with INVOX_FORCE_TTY=1, want true")
	}
}

func TestCanPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stdinTTY    bool
		stderrTTY   bool
		neverPrompt bool
		want        bool
	}{
		{name: "no terminal", want: false},
		{name: "stdin only", stdinTTY: true, want: false},
		{name: "stderr only", stderrTTY: true, want: false},
		{name: "stdin and stderr", stdinTTY: true, stderrTTY: true, want: true},
		{name: "prompting disabled", stdinTTY: true, stderrTTY: true, neverPrompt: true, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ios, _, _, _ := Test()
			ios.SetStdinTTY(tc.stdinTTY)
			ios.SetStderrTTY(tc.stderrTTY)
			ios.SetNeverPrompt(tc.neverPrompt)
			if got := ios.CanPrompt(); got != tc.want {
				t.Fatalf("CanPrompt() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTestStreamsAreBuffers(t *testing.T) {
	t.Parallel()

	ios, in, out, errOut := Test()
	in.WriteString("answer\n")
	if _, err := io.WriteString(ios.Out, "data\n"); err != nil {
		t.Fatalf("write Out: %v", err)
	}
	if _, err := io.WriteString(ios.ErrOut, "status\n"); err != nil {
		t.Fatalf("write ErrOut: %v", err)
	}
	got, err := io.ReadAll(ios.In)
	if err != nil {
		t.Fatalf("read In: %v", err)
	}

	if string(got) != "answer\n" || out.String() != "data\n" || errOut.String() != "status\n" {
		t.Fatalf("In=%q Out=%q ErrOut=%q, want %q %q %q", got, out, errOut, "answer\n", "data\n", "status\n")
	}
	if ios.IsStdinTTY() || ios.IsStdoutTTY() || ios.IsStderrTTY() {
		t.Fatal("Test() streams report a terminal, want none")
	}
}
