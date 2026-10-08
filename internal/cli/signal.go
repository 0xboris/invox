package cli

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/0xboris/invox/internal/cli/cmdutil"
)

// SignalError is the cancel cause when a signal ended the run. exitCode maps
// it to 128 plus the signal number.
type SignalError struct {
	Signal syscall.Signal
}

func (e *SignalError) Error() string { return "signal: " + e.Signal.String() }

// signalContext returns a context that is cancelled with a *SignalError on the
// first SIGINT or SIGTERM. After that signal the default handling returns, so
// a second Ctrl-C ends invox at once. SIGINT is ignored while
// cmdutil.HoldInterrupt holds it.
func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	held := new(atomic.Int32)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer signal.Stop(signals)
		for {
			select {
			case sig := <-signals:
				if sig == os.Interrupt && held.Load() > 0 {
					continue
				}
				cancel(&SignalError{Signal: sig.(syscall.Signal)})
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return cmdutil.WithInterruptHold(ctx, held), func() {
		signal.Stop(signals)
		cancel(nil)
	}
}
