package cmdutil

import (
	"context"
	"sync/atomic"
)

type interruptHoldKey struct{}

// WithInterruptHold returns ctx carrying held, the count of HoldInterrupt
// calls not yet released. The signal handler ignores SIGINT while it is
// above zero.
func WithInterruptHold(ctx context.Context, held *atomic.Int32) context.Context {
	return context.WithValue(ctx, interruptHoldKey{}, held)
}

// HoldInterrupt makes the signalContext that ctx comes from ignore SIGINT
// until release is called. A terminal sends Ctrl-C to every program in the
// foreground, so an editor invox waits on gets it too and decides what it
// means, as with git. SIGTERM, which only invox receives, still cancels ctx.
func HoldInterrupt(ctx context.Context) (release func()) {
	held, ok := ctx.Value(interruptHoldKey{}).(*atomic.Int32)
	if !ok {
		return func() {}
	}
	held.Add(1)
	return func() { held.Add(-1) }
}
