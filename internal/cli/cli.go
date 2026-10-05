package cli

import (
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/iostreams"
)

func Main(args []string, ios *iostreams.IOStreams) int {
	if len(args) == 0 {
		return rootUsageError(ios, "missing subcommand")
	}

	if isHelpToken(args[0]) {
		printRootHelp(ios.Out)
		return 0
	}

	if args[0] == "--version" {
		return runVersion(ios, args[1:])
	}

	if args[0] == "help" {
		return runHelp(ios, args[1:])
	}

	switch args[0] {
	case "customer":
		return runCustomer(ios, args[1:])
	case "config":
		return runConfig(ios, args[1:])
	case "init":
		return runInit(ios, args[1:])
	case "template":
		return runTemplate(ios, args[1:])
	case "completion":
		return runCompletion(ios, args[1:])
	case "new":
		return runNew(ios, args[1:])
	case "increment":
		return runIncrement(ios, args[1:])
	case "validate":
		return runValidate(ios, args[1:])
	case "render":
		return runRender(ios, args[1:])
	case "email":
		return runEmail(ios, args[1:])
	case "send":
		return runEmail(ios, args[1:])
	case "build":
		return runBuild(ios, args[1:])
	case "archive":
		return runArchive(ios, args[1:])
	case "version":
		return runVersion(ios, args[1:])
	default:
		return rootUsageError(ios, fmt.Sprintf("unknown subcommand %q", args[0]))
	}
}

func runHelp(ios *iostreams.IOStreams, args []string) int {
	if len(args) == 0 {
		printRootHelp(ios.Out)
		return 0
	}

	if args[0] == "customer" {
		if len(args) == 1 {
			printCustomerHelp(ios.Out)
			return 0
		}
		if len(args) == 2 && (args[1] == "list" || args[1] == "config") {
			spec, _ := lookupCommand("customer " + args[1])
			printCommandHelp(ios.Out, spec)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "template" {
		if len(args) == 1 {
			printTemplateHelp(ios.Out)
			return 0
		}
		if len(args) == 2 && args[1] == "list" {
			printTemplateListHelp(ios.Out)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "completion" {
		if len(args) == 1 || (len(args) == 2 && args[1] == "zsh") {
			printCompletionHelp(ios.Out)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "customers" {
		if len(args) == 1 {
			printCustomersHelp(ios.Out)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "issuer" {
		if len(args) == 1 {
			printIssuerHelp(ios.Out)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "defaults" || args[0] == "invoice-defaults" || args[0] == "invoice_defaults" {
		if len(args) == 1 {
			printDefaultsHelp(ios.Out)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "version" && len(args) == 1 {
		printVersionHelp(ios.Out)
		return 0
	}

	if args[0] == "archive" {
		if len(args) == 1 {
			spec, _ := lookupCommand("archive")
			printCommandHelp(ios.Out, spec)
			return 0
		}
		if len(args) == 2 && (args[1] == "edit" || args[1] == "list") {
			spec, _ := lookupCommand("archive " + args[1])
			printCommandHelp(ios.Out, spec)
			return 0
		}
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	spec, ok := lookupCommand(strings.Join(args, " "))
	if !ok {
		return rootUsageError(ios, fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}
	printCommandHelp(ios.Out, spec)
	return 0
}

func runCustomer(ios *iostreams.IOStreams, args []string) int {
	if len(args) == 0 {
		printCustomerHelp(ios.Out)
		return 0
	}
	if len(args) == 1 && wantsHelp(args) {
		printCustomerHelp(ios.Out)
		return 0
	}

	switch args[0] {
	case "list":
		return runCustomerList(ios, args[1:])
	case "config":
		return runCustomerConfig(ios, args[1:])
	default:
		return customerUsageError(ios, fmt.Sprintf("unknown customer subcommand %q", args[0]))
	}
}

func rootUsageError(ios *iostreams.IOStreams, message string) int {
	fmt.Fprintf(ios.ErrOut, "error: %s\n\n", message)
	printRootHelp(ios.ErrOut)
	return 2
}

func customerUsageError(ios *iostreams.IOStreams, message string) int {
	fmt.Fprintf(ios.ErrOut, "error: %s\n\n", message)
	printCustomerHelp(ios.ErrOut)
	return 2
}

func templateUsageError(ios *iostreams.IOStreams, message string) int {
	fmt.Fprintf(ios.ErrOut, "error: %s\n\n", message)
	printTemplateHelp(ios.ErrOut)
	return 2
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if isHelpToken(arg) {
			return true
		}
	}
	return false
}

func isHelpToken(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "-help"
}
