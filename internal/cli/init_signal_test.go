package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
)

func TestSignalAtInitLegacyPrompt(t *testing.T) {
	// Ctrl-C at the prompt is a quiet cancel; SIGTERM still ends invox as 143.
	for _, tc := range []struct {
		name     string
		signal   syscall.Signal
		exitCode int
	}{
		{name: "SIGINT", signal: syscall.SIGINT, exitCode: 2},
		{name: "SIGTERM", signal: syscall.SIGTERM, exitCode: 143},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newConfigLayout(t)
			if err := os.Remove(l.invoxDir); err != nil {
				t.Fatalf("Remove returned error: %v", err)
			}
			writeTestFile(t, filepath.Join(l.legacyDir, "customers.yaml"), "legacy\n")
			ios := promptStreams(true, "")
			stdin, unanswered := io.Pipe()
			t.Cleanup(func() { _ = unanswered.Close() })
			ios.In = stdin
			isolateUserDirs(t)
			f := factory.New(ios, run.Exec{}, env.System())
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(&SignalError{Signal: tc.signal})

			exitCode := make(chan int, 1)
			go func() { exitCode <- mainContext(ctx, []string{"init"}, f) }()
			var got int
			select {
			case got = <-exitCode:
			case <-time.After(30 * time.Second):
				t.Fatal("invox kept waiting for an answer after the signal")
			}

			if got != tc.exitCode {
				t.Fatalf("exitCode = %d, want %d", got, tc.exitCode)
			}
			if stdout := ios.Out.(*bytes.Buffer).String(); stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			want := "Copy customers.yaml from " + l.legacyDir + " to " + l.invoxDir + "? [y/N] \n"
			if stderr := ios.ErrOut.(*bytes.Buffer).String(); stderr != want {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
			if _, err := os.Stat(l.invoxDir); !os.IsNotExist(err) {
				t.Fatalf("config directory %s exists after the signal (Stat error %v), want nothing written", l.invoxDir, err)
			}
		})
	}
}
