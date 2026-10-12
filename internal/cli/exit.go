package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

// exitCode reports err, which cmd returned, on f's stderr and returns the
// code invox exits with. It is the only place that maps errors to exit codes.
func exitCode(f *cmdutil.Factory, cmd *cobra.Command, err error) int {
	var flagErr *cmdutil.FlagError
	var sigErr *SignalError
	switch {
	case err == nil, err == flag.ErrHelp:
		return 0
	case errors.Is(err, cmdutil.SilentError):
		return 1
	case errors.Is(err, cmdutil.CancelError):
		return 2
	case errors.As(err, &sigErr):
		return 128 + int(sigErr.Signal)
	case errors.As(err, &flagErr):
		if flagErr.Root {
			cmd = cmd.Root()
		}
		printError(f, fmt.Sprintf("%s\nRun '%s --help' for usage.", flagErr.Err, cmd.CommandPath()))
		return 2
	}
	message := err.Error()
	if hint := errorHint(err, f.ConfigFile); hint != "" {
		message += "\n" + hint
	}
	printError(f, message)
	return 1
}

// printError prints message on f's stderr with the error prefix, and with
// paths under f's working directory made relative to it. A path counts only
// where it starts: at the start of the message or after a space, quote or
// parenthesis.
func printError(f *cmdutil.Factory, message string) {
	if cwd, err := f.Env.Getwd(); err == nil && filepath.Dir(cwd) != cwd {
		pathStart := regexp.MustCompile("(^|[\\s'\"`(])" + regexp.QuoteMeta(cwd+string(filepath.Separator)))
		message = pathStart.ReplaceAllString(message, "${1}")
	}
	fmt.Fprintf(f.IOStreams.ErrOut, "error: %s\n", message)
}

// errorHint returns the next step that fixes err, or "" when there is none.
// configFile is the --config file, which the config hint must name: `invox
// config` alone opens the default one.
func errorHint(err error, configFile string) string {
	var unknownCustomer *invoice.UnknownCustomerError
	var configErr *billing.ConfigError
	var duplicate *invoice.DuplicateInvoiceNumberError
	var toolMissing *billing.ToolMissingError
	switch {
	case errors.As(err, &configErr) && configFile != "":
		return fmt.Sprintf("Run '%s --config %s config' to open and fix the config file.", commandName, configFile)
	case errors.As(err, &unknownCustomer):
		return fmt.Sprintf("Run '%s customer list' to see the customer IDs.", commandName)
	case len(unknownKeyHelpTopics(err)) > 0:
		var commands []string
		for _, topic := range unknownKeyHelpTopics(err) {
			commands = append(commands, fmt.Sprintf("'%s help %s'", commandName, topic))
		}
		return fmt.Sprintf("Remove the unknown keys or fix their spelling; %s lists the supported fields.", strings.Join(commands, " or "))
	case errors.As(err, &configErr):
		return fmt.Sprintf("Run '%s config' to open and fix the config file.", commandName)
	case errors.As(err, &duplicate):
		return fmt.Sprintf("Run '%s increment -i %s' to give it the next free number, then archive it again.", commandName, duplicate.InvoicePath)
	case errors.As(err, &toolMissing):
		return toolMissing.Hint
	}
	return ""
}

// unknownKeyHelpTopics returns the help topics that document the files in
// which err reports unknown keys, in the order the files first appear.
func unknownKeyHelpTopics(err error) []string {
	topicOf := map[string]string{"customers": "customers", "issuer": "issuer", "invoice": "defaults"}
	var topics []string
	seen := map[string]bool{}
	var walk func(error)
	walk = func(err error) {
		switch e := err.(type) {
		case *billing.DecodeError:
			if topic := topicOf[e.Schema]; e.UnknownKey && topic != "" && !seen[topic] {
				seen[topic] = true
				topics = append(topics, topic)
			}
		case interface{ Unwrap() []error }:
			for _, inner := range e.Unwrap() {
				walk(inner)
			}
		case interface{ Unwrap() error }:
			walk(e.Unwrap())
		}
	}
	walk(err)
	return topics
}
