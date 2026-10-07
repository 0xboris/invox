package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// hostOS selects the platform-specific install hints.
var hostOS = runtime.GOOS

// exitCode reports err on ios.ErrOut and returns the code invox exits with.
// It is the only place that maps errors to exit codes.
func exitCode(ios *iostreams.IOStreams, err error) int {
	var flagErr *cmdutil.FlagError
	switch {
	case err == nil, err == flag.ErrHelp:
		return 0
	case errors.Is(err, cmdutil.SilentError):
		return 1
	case errors.Is(err, cmdutil.CancelError):
		return 2
	case errors.As(err, &flagErr):
		command := commandName
		if flagErr.Command != "" {
			command += " " + flagErr.Command
		}
		printError(ios.ErrOut, fmt.Sprintf("%s\nRun '%s --help' for usage.", flagErr.Err, command))
		return 2
	}
	message := err.Error()
	if hint := errorHint(err); hint != "" {
		message += "\n" + hint
	}
	printError(ios.ErrOut, message)
	return 1
}

// printError prints message with the error prefix, and with paths under the
// working directory made relative to it. A path counts only where it starts:
// at the start of the message or after a space, quote or parenthesis.
func printError(w io.Writer, message string) {
	if cwd, err := os.Getwd(); err == nil && filepath.Dir(cwd) != cwd {
		pathStart := regexp.MustCompile("(^|[\\s'\"`(])" + regexp.QuoteMeta(cwd+string(filepath.Separator)))
		message = pathStart.ReplaceAllString(message, "${1}")
	}
	fmt.Fprintf(w, "error: %s\n", message)
}

// errorHint returns the next step that fixes err, or "" when there is none.
func errorHint(err error) string {
	var unknownCustomer *invoice.UnknownCustomerError
	var configErr *invoice.ConfigError
	var duplicate *invoice.DuplicateInvoiceNumberError
	switch {
	case errors.As(err, &unknownCustomer):
		return fmt.Sprintf("Run '%s customer list' to see the customer IDs.", commandName)
	case errors.As(err, &configErr):
		return fmt.Sprintf("Run '%s config' to open and fix the config file.", commandName)
	case errors.As(err, &duplicate):
		return fmt.Sprintf("Run '%s increment -i %s' to give it the next free number, then archive it again.", commandName, duplicate.InvoicePath)
	case errors.Is(err, invoice.ErrTectonicNotFound):
		return tectonicInstallHint(hostOS)
	}
	return ""
}

func tectonicInstallHint(goos string) string {
	if goos == "darwin" {
		return "Install it with 'brew install tectonic', then rerun this command."
	}
	return "Install it from https://tectonic-typesetting.github.io, then rerun this command."
}
