package cli

import (
	"fmt"
	"os"
	"strings"
)

func Run(args []string) int {
	if len(args) == 0 {
		return rootUsageError("missing subcommand")
	}

	if isHelpToken(args[0]) {
		printRootHelp(os.Stdout)
		return 0
	}

	if args[0] == "help" {
		return runHelp(args[1:])
	}

	switch args[0] {
	case "customer":
		return runCustomer(args[1:])
	case "config":
		return runConfig(args[1:])
	case "init":
		return runInit(args[1:])
	case "new":
		return runNew(args[1:])
	case "increment":
		return runIncrement(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "render":
		return runRender(args[1:])
	case "build":
		return runBuild(args[1:])
	case "archive":
		return runArchive(args[1:])
	default:
		return rootUsageError(fmt.Sprintf("unknown subcommand %q", args[0]))
	}
}

func runHelp(args []string) int {
	if len(args) == 0 {
		printRootHelp(os.Stdout)
		return 0
	}

	if args[0] == "customer" {
		if len(args) == 1 {
			printCustomerHelp(os.Stdout)
			return 0
		}
		if len(args) == 2 && (args[1] == "list" || args[1] == "config") {
			spec, _ := lookupCommand("customer " + args[1])
			printCommandHelp(os.Stdout, spec)
			return 0
		}
		return rootUsageError(fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	if args[0] == "archive" {
		if len(args) == 1 {
			spec, _ := lookupCommand("archive")
			printCommandHelp(os.Stdout, spec)
			return 0
		}
		if len(args) == 2 && (args[1] == "edit" || args[1] == "list") {
			spec, _ := lookupCommand("archive " + args[1])
			printCommandHelp(os.Stdout, spec)
			return 0
		}
		return rootUsageError(fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}

	spec, ok := lookupCommand(strings.Join(args, " "))
	if !ok {
		return rootUsageError(fmt.Sprintf("unknown help topic %q", strings.Join(args, " ")))
	}
	printCommandHelp(os.Stdout, spec)
	return 0
}

func runCustomer(args []string) int {
	if len(args) == 0 {
		printCustomerHelp(os.Stdout)
		return 0
	}
	if len(args) == 1 && wantsHelp(args) {
		printCustomerHelp(os.Stdout)
		return 0
	}

	switch args[0] {
	case "list":
		return runCustomerList(args[1:])
	case "config":
		return runCustomerConfig(args[1:])
	default:
		return customerUsageError(fmt.Sprintf("unknown customer subcommand %q", args[0]))
	}
}

func rootUsageError(message string) int {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", message)
	printRootHelp(os.Stderr)
	return 2
}

func customerUsageError(message string) int {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", message)
	printCustomerHelp(os.Stderr)
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
