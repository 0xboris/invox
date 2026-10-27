package proc

// SignalError is the cancel cause when a signal ended the run. The CLI exits
// with 128 plus Number.
type SignalError struct {
	Number int
	// Name is the signal's description, such as "interrupt".
	Name string
	// Interrupt is set for SIGINT, the signal Ctrl-C sends.
	Interrupt bool
}

func (e *SignalError) Error() string { return "signal: " + e.Name }
