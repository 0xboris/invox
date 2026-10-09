package cli

import (
	"context"
	"errors"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
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
	if len(args) == 0 || (args[0] != cobra.ShellCompRequestCmd && args[0] != cobra.ShellCompNoDescRequestCmd) {
		// A completion request passes the words typed so far as they are.
		if err := checkSingleDashFlags(root, args); err != nil {
			return exitCode(f.IOStreams, err)
		}
		args = versionFlagToCommand(args)
	}
	root.SetArgs(args)
	_, err := root.ExecuteContextC(ctx)
	if err == nil {
		err = helpErr()
	}
	var configErr *billing.ConfigError
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
