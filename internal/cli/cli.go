package cli

import (
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

func Main(args []string, ios *iostreams.IOStreams, e env.Env) int {
	return exitCode(ios, run(ios, e, args))
}

func run(ios *iostreams.IOStreams, e env.Env, args []string) error {
	if len(args) == 0 {
		return cmdutil.FlagErrorf("", "missing subcommand")
	}

	if isHelpToken(args[0]) {
		printRootHelp(ios.Out, userHost(e))
		return nil
	}

	if args[0] == "--version" {
		return runVersion(ios, args[1:])
	}

	if args[0] == "help" {
		return runHelp(ios, e, args[1:])
	}

	switch args[0] {
	case "customer":
		return runCustomer(ios, e, args[1:])
	case "config":
		return runConfig(ios, e, args[1:])
	case "init":
		return runInit(ios, e, args[1:])
	case "template":
		return runTemplate(ios, e, args[1:])
	case "completion":
		return runCompletion(ios, args[1:])
	case "new":
		return runNew(ios, e, args[1:])
	case "increment":
		return runIncrement(ios, e, args[1:])
	case "validate":
		return runValidate(ios, e, args[1:])
	case "render":
		return runRender(ios, e, args[1:])
	case "email":
		return runEmail(ios, e, args[1:])
	case "send":
		return runEmail(ios, e, args[1:])
	case "build":
		return runBuild(ios, e, args[1:])
	case "archive":
		return runArchive(ios, e, args[1:])
	case "version":
		return runVersion(ios, args[1:])
	default:
		return cmdutil.FlagErrorf("", "unknown subcommand %q", args[0])
	}
}

func runHelp(ios *iostreams.IOStreams, e env.Env, args []string) error {
	h := userHost(e)
	if len(args) == 0 {
		printRootHelp(ios.Out, h)
		return nil
	}

	if args[0] == "customer" {
		if len(args) == 1 {
			printCustomerHelp(ios.Out, h)
			return nil
		}
		if len(args) == 2 && (args[1] == "list" || args[1] == "config") {
			spec, _ := lookupCommand("customer " + args[1])
			printCommandHelp(ios.Out, h, spec)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if args[0] == "template" {
		if len(args) == 1 {
			printTemplateHelp(ios.Out)
			return nil
		}
		if len(args) == 2 && args[1] == "list" {
			printTemplateListHelp(ios.Out)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if args[0] == "completion" {
		if len(args) == 1 || (len(args) == 2 && args[1] == "zsh") {
			printCompletionHelp(ios.Out)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if args[0] == "customers" {
		if len(args) == 1 {
			printCustomersHelp(ios.Out, h)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if args[0] == "issuer" {
		if len(args) == 1 {
			printIssuerHelp(ios.Out, h)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if args[0] == "defaults" || args[0] == "invoice-defaults" || args[0] == "invoice_defaults" {
		if len(args) == 1 {
			printDefaultsHelp(ios.Out, h)
			return nil
		}
		return unknownHelpTopic(args)
	}

	if len(args) == 1 && args[0] == "environment" {
		printEnvironmentHelp(ios.Out, h)
		return nil
	}

	if len(args) == 1 && args[0] == "exit-codes" {
		printExitCodesHelp(ios.Out)
		return nil
	}

	if args[0] == "version" && len(args) == 1 {
		printVersionHelp(ios.Out)
		return nil
	}

	if args[0] == "archive" {
		if len(args) == 1 {
			spec, _ := lookupCommand("archive")
			printCommandHelp(ios.Out, h, spec)
			return nil
		}
		if len(args) == 2 && (args[1] == "edit" || args[1] == "list") {
			spec, _ := lookupCommand("archive " + args[1])
			printCommandHelp(ios.Out, h, spec)
			return nil
		}
		return unknownHelpTopic(args)
	}

	spec, ok := lookupCommand(strings.Join(args, " "))
	if !ok {
		return unknownHelpTopic(args)
	}
	printCommandHelp(ios.Out, h, spec)
	return nil
}

func runCustomer(ios *iostreams.IOStreams, e env.Env, args []string) error {
	h := userHost(e)
	if len(args) == 0 {
		printCustomerHelp(ios.Out, h)
		return nil
	}
	if len(args) == 1 && wantsHelp(args) {
		printCustomerHelp(ios.Out, h)
		return nil
	}

	switch args[0] {
	case "list":
		return runCustomerList(ios, e, args[1:])
	case "config":
		return runCustomerConfig(ios, e, args[1:])
	default:
		return cmdutil.FlagErrorf("customer", "unknown customer subcommand %q", args[0])
	}
}

func unknownHelpTopic(args []string) error {
	return cmdutil.FlagErrorf("", "unknown help topic %q", strings.Join(args, " "))
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
