package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/config"
)

const commandName = "invox"

func Main(args []string, f *cmdutil.Factory) int {
	cobra.MousetrapHelpText = ""
	ctx, stop := signalContext(context.Background())
	defer stop()
	return mainContext(ctx, args, f)
}

// mainContext runs the command under ctx. When a signal cancelled ctx, the
// command's error becomes the signal's, except that Ctrl-C at a prompt stays
// a quiet cancel.
//
// A terminal Ctrl-C also reaches the child program, so the command can fail
// before ctx is cancelled. That ordering is not expected in practice: the
// signal goroutine needs a few scheduler hand-offs, while the command first
// waits for the child to exit and be reaped. If it ever happened, invox would
// exit 1 and print the child's error instead of exiting 130.
func mainContext(ctx context.Context, args []string, f *cmdutil.Factory) int {
	if f.Env.Getenv("INVOX_PROMPT_DISABLED") != "" {
		f.IOStreams.SetNeverPrompt(true)
	}
	root, helpErr := newRootCmd(f)
	root.SetArgs(versionFlagToCommand(normalizeLongFlags(root, args, f.IOStreams.ErrOut)))
	_, err := root.ExecuteContextC(ctx)
	if err == nil {
		err = helpErr()
	}
	warnLegacyFiles(f)
	var configErr *config.Error
	if f.ConfigFile != "" && errors.As(err, &configErr) {
		err = &configFlagError{err: err, path: f.ConfigFile}
	}
	var sigErr *SignalError
	if err != nil && errors.As(context.Cause(ctx), &sigErr) &&
		!(sigErr.Signal == syscall.SIGINT && errors.Is(err, cmdutil.CancelError)) {
		err = sigErr
	}
	return exitCode(f.IOStreams, err)
}

// warnLegacyFiles prints one line when the command read files from the
// deprecated config directory.
func warnLegacyFiles(f *cmdutil.Factory) {
	h := f.Host()
	used := h.LegacyFilesUsed()
	if len(used) == 0 {
		return
	}
	legacyDir := h.LegacyConfigDir()
	names := make([]string, len(used))
	for i, path := range used {
		names[i] = path
		if rel, err := filepath.Rel(legacyDir, path); err == nil {
			names[i] = rel
		}
	}
	list, pronoun := names[0], "it"
	if len(names) > 1 {
		list, pronoun = strings.Join(names[:len(names)-1], ", ")+" and "+names[len(names)-1], "them"
	}
	fmt.Fprintf(f.IOStreams.ErrOut, "warning: using %s from deprecated config directory %s; run '%s init' to copy %s to %s\n", list, legacyDir, commandName, pronoun, h.ConfigDir())
}

func runHelp(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
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

func unknownHelpTopic(args []string) error {
	return cmdutil.FlagErrorf("", "unknown help topic %q", strings.Join(args, " "))
}
