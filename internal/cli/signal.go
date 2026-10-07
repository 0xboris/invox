package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SignalError is the cancel cause when a signal ended the run. exitCode maps
// it to 128 plus the signal number.
type SignalError struct {
	Signal syscall.Signal
}

func (e *SignalError) Error() string { return "signal: " + e.Signal.String() }

// signalContext returns a context that is cancelled with a *SignalError on the
// first SIGINT or SIGTERM. After that signal the default handling returns, so
// a second Ctrl-C ends invox at once.
func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-signals:
			cancel(&SignalError{Signal: sig.(syscall.Signal)})
		case <-ctx.Done():
		}
		signal.Stop(signals)
	}()
	return ctx, func() {
		signal.Stop(signals)
		cancel(nil)
	}
}
