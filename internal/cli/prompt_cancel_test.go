package cli

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
)

func TestSignalAtReplacePrompt(t *testing.T) {
	// Ctrl-C at a prompt is a quiet cancel; SIGTERM still ends invox as 143.
	for _, tc := range []struct {
		name     string
		signal   syscall.Signal
		exitCode int
	}{
		{name: "SIGINT", signal: syscall.SIGINT, exitCode: 2},
		{name: "SIGTERM", signal: syscall.SIGTERM, exitCode: 143},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := setupEditedArchive(t)
			ios := promptStreams(true, "")
			stdin, unanswered := io.Pipe()
			t.Cleanup(func() { _ = unanswered.Close() })
			ios.In = stdin
			f := factory.New(ios, run.Exec{}, env.System())
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(&SignalError{Signal: tc.signal})

			exitCode := make(chan int, 1)
			go func() { exitCode <- mainContext(ctx, []string{"archive", "add", "first.yaml"}, f) }()
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
			want := "Replace archived invoice " + e.archivedPath + "? The previous version is kept in " +
				filepath.Join(e.archiveDir, ".history") + ". [y/N] \n"
			if stderr := ios.ErrOut.(*bytes.Buffer).String(); stderr != want {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
			e.assertUnchanged(t)
		})
	}
}
