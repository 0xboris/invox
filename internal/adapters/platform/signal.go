package platform

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/0xboris/invox/internal/utilities/proc"
)

// SignalContext returns a context that is cancelled with a *proc.SignalError on the
// first SIGINT or SIGTERM. After that signal the default handling returns, so
// a second Ctrl-C ends invox at once. SIGINT is ignored while
// proc.HoldInterrupt holds it.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
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
				cancel(NewSignalError(sig))
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return proc.WithInterruptHold(ctx, held), func() {
		signal.Stop(signals)
		cancel(nil)
	}
}

// NewSignalError returns the cancel cause for sig.
func NewSignalError(sig os.Signal) *proc.SignalError {
	number := sig.(syscall.Signal)
	return &proc.SignalError{Number: int(number), Name: number.String(), Interrupt: sig == os.Interrupt}
}
