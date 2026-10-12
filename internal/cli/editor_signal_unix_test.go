//go:build unix

package cli_test

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/clitest"
)

// guardSignals keeps SIGINT and SIGTERM caught for the whole test, so a
// signal the test sends to itself after signalContext stopped listening
// cannot end the test binary.
func guardSignals(t *testing.T) {
	t.Helper()

	guard := make(chan os.Signal, 64)
	signal.Notify(guard, os.Interrupt, syscall.SIGTERM)
	t.Cleanup(func() { signal.Stop(guard) })
}

func kill(t *testing.T, sig syscall.Signal) {
	t.Helper()

	if err := syscall.Kill(os.Getpid(), sig); err != nil {
		t.Fatalf("kill(self, %v) returned error: %v", sig, err)
	}
}

// signalUntilDone sends sig to the test process until ctx is done. A signal
// that arrives while signalContext has an unread one is dropped, so one send
// is not enough.
func signalUntilDone(t *testing.T, ctx context.Context, sig syscall.Signal) {
	t.Helper()

	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		kill(t, sig)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("context not cancelled within 10s of %v", sig)
		}
	}
}

func TestCtrlCWhileTheEditorRunsReachesOnlyTheEditor(t *testing.T) {
	x := clitest.New(t)

	guardSignals(t)

	x.IO.SetStdinTTY(true)
	x.IO.SetStderrTTY(true)
	ctx, stop := cli.SignalContext(context.Background())
	defer stop()

	// The editor sees Ctrl-C, then SIGTERM. Only SIGTERM may cancel the run,
	// which run.Exec turns into a SIGTERM for the editor.
	x.Stub.Register("vi", func(run.Cmd) error {
		kill(t, syscall.SIGINT)
		signalUntilDone(t, ctx, syscall.SIGTERM)
		return &run.ExecError{Name: "vi", Code: -1, Err: errors.New("signal: terminated")}
	})

	exitCode := cli.MainContext(ctx, []string{"config"}, x.Factory())

	var sigErr *cli.SignalError
	if !errors.As(context.Cause(ctx), &sigErr) || sigErr.Signal != syscall.SIGTERM {
		t.Fatalf("cancel cause = %v, want SIGTERM", context.Cause(ctx))
	}
	if exitCode != 143 {
		t.Fatalf("exitCode = %d, want 143", exitCode)
	}
}

func TestCtrlCCancelsAgainAfterTheEditorExits(t *testing.T) {
	guardSignals(t)
	ctx, stop := cli.SignalContext(context.Background())
	defer stop()

	release := cmdutil.HoldInterrupt(ctx)
	release()
	signalUntilDone(t, ctx, syscall.SIGINT)

	var sigErr *cli.SignalError
	if !errors.As(context.Cause(ctx), &sigErr) || sigErr.Signal != syscall.SIGINT {
		t.Fatalf("cancel cause = %v, want SIGINT", context.Cause(ctx))
	}
}

func TestCtrlCCancelsAgainAfterOpenInEditorReturns(t *testing.T) {
	x := clitest.New(t)

	guardSignals(t)
	x.ExpectEditor(nil)
	f := x.Factory()
	ctx, stop := cli.SignalContext(context.Background())
	defer stop()

	if err := cmdutil.OpenInEditor(ctx, f.IOStreams, f.Editor, filepath.Join(t.TempDir(), "config.yaml"), "edit it"); err != nil {
		t.Fatalf("OpenInEditor returned error: %v", err)
	}
	signalUntilDone(t, ctx, syscall.SIGINT)

	var sigErr *cli.SignalError
	if !errors.As(context.Cause(ctx), &sigErr) || sigErr.Signal != syscall.SIGINT {
		t.Fatalf("cancel cause = %v, want SIGINT", context.Cause(ctx))
	}
}
