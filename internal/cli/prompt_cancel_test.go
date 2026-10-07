package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
)

func TestSignalAtReplacePromptExits2Quietly(t *testing.T) {
	e := setupEditedArchive(t)
	ios := promptStreams(true, "")
	stdin, unanswered := io.Pipe()
	t.Cleanup(func() { unanswered.Close() })
	ios.In = stdin
	f := cmdutil.NewFactory(ios, run.Exec{}, runtime.GOOS, os.Getenv)
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(&SignalError{Signal: syscall.SIGINT})

	exitCode := make(chan int, 1)
	go func() { exitCode <- mainContext(ctx, []string{"archive", "first.yaml"}, f) }()
	var got int
	select {
	case got = <-exitCode:
	case <-time.After(30 * time.Second):
		t.Fatal("invox kept waiting for an answer after the signal")
	}

	if got != 2 {
		t.Fatalf("exitCode = %d, want 2", got)
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
}
