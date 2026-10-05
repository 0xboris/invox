package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// exitCode reports err on ios.ErrOut and returns the code invox exits with.
// It is the only place that maps errors to exit codes.
func exitCode(ios *iostreams.IOStreams, err error) int {
	var flagErr *cmdutil.FlagError
	var execErr *cmdutil.ExecError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, cmdutil.SilentError):
		return 1
	case errors.Is(err, cmdutil.CancelError):
		return 2
	case errors.As(err, &execErr):
		if execErr.Code > 0 {
			return execErr.Code
		}
		return 1
	case errors.As(err, &flagErr):
		printUsageError(ios.ErrOut, flagErr)
		return 2
	}
	printRuntimeError(ios.ErrOut, err)
	return 1
}

func printUsageError(w io.Writer, err *cmdutil.FlagError) {
	switch err.Command {
	case "":
		fmt.Fprintf(w, "error: %s\n\n", err.Err)
		printRootHelp(w)
	case "customer":
		fmt.Fprintf(w, "error: %s\n\n", err.Err)
		printCustomerHelp(w)
	case "template":
		fmt.Fprintf(w, "error: %s\n\n", err.Err)
		printTemplateHelp(w)
	default:
		spec, _ := lookupCommand(err.Command)
		printCommandError(w, spec, err.Err.Error())
	}
}

func printRuntimeError(w io.Writer, err error) {
	fmt.Fprintln(w, err)
	var configErr *invoice.ConfigError
	if errors.As(err, &configErr) {
		fmt.Fprintf(w, "Run `%s config` to open and fix the config file.\n", commandName)
	}
	var duplicate *invoice.DuplicateInvoiceNumberError
	if errors.As(err, &duplicate) {
		baseDir, _ := os.Getwd()
		fmt.Fprintf(
			w,
			"Run `invox increment -i %s` to give it the next free number, then archive it again.\n",
			invoice.DisplayPath(duplicate.InvoicePath, baseDir),
		)
	}
}
