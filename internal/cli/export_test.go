package cli

// The tests in package cli_test run invox through clitest, as the command
// tests do. The signal tests also run it under a context they cancel.
var (
	MainContext   = mainContext
	SignalContext = signalContext
)
