package cli

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// SignalError is the cancel cause when a signal ended the run. exitCode maps
// it to 128 plus the signal number.
type SignalError struct {
	Signal syscall.Signal
}

func (e *SignalError) Error() string { return "signal: " + e.Signal.String() }

type interruptHoldKey struct{}

// signalContext returns a context that is cancelled with a *SignalError on the
// first SIGINT or SIGTERM. After that signal the default handling returns, so
// a second Ctrl-C ends invox at once. SIGINT is ignored while holdInterrupt
// holds it.
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
	return context.WithValue(ctx, interruptHoldKey{}, held), func() {
		signal.Stop(signals)
		cancel(nil)
	}
}

// holdInterrupt makes the signalContext that ctx comes from ignore SIGINT
// until release is called. A terminal sends Ctrl-C to every program in the
// foreground, so an editor invox waits on gets it too and decides what it
// means, as with git. SIGTERM, which only invox receives, still cancels ctx.
func holdInterrupt(ctx context.Context) (release func()) {
	held, ok := ctx.Value(interruptHoldKey{}).(*atomic.Int32)
	if !ok {
		return func() {}
	}
	held.Add(1)
	return func() { held.Add(-1) }
}
