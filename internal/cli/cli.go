package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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
		args = versionFlagToCommand(normalizeLongFlags(root, args, f.IOStreams.ErrOut))
	}
	root.SetArgs(args)
	_, err := root.ExecuteContextC(ctx)
	if err == nil {
		err = helpErr()
	}
	warnLegacyFiles(f)
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

// warnLegacyFiles prints one line when the command read files from the
// deprecated config directory.
func warnLegacyFiles(f *cmdutil.Factory) {
	svc := f.Service(cmdutil.Files{})
	used := svc.LegacyFilesUsed()
	if len(used) == 0 {
		return
	}
	locations := svc.Locations()
	legacyDir := locations.LegacyDir
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
	fmt.Fprintf(f.IOStreams.ErrOut, "warning: using %s from deprecated config directory %s; run '%s init' to copy %s to %s\n", list, legacyDir, commandName, pronoun, locations.ConfigDir)
}
