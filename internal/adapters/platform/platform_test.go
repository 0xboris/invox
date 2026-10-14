package platform

import (
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

	ios := newStreams(devNull, devNull, devNull)
	if ios.IsStdinTTY() || ios.IsStderrTTY() {
		t.Fatalf("IsStdinTTY() = %v, IsStderrTTY() = %v on %s, want false", ios.IsStdinTTY(), ios.IsStderrTTY(), os.DevNull)
	}
	if ios.CanPrompt() {
		t.Fatalf("CanPrompt() = true on %s, want false", os.DevNull)
	}
}

func TestStreamsForceTTY(t *testing.T) {
	t.Setenv("INVOX_FORCE_TTY", "1")

	if !Streams().IsStdoutTTY() {
		t.Fatal("IsStdoutTTY() = false with INVOX_FORCE_TTY=1, want true")
	}
}
