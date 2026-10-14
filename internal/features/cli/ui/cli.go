package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/domain/config"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/proc"
)

const commandName = "invox"

// Main runs the invox command line args under ctx, with commands below the
// root, and returns the exit code. When a signal cancelled ctx, the
// command's error becomes the signal's, except that Ctrl-C at a prompt stays
// a quiet cancel.
//
// A terminal Ctrl-C also reaches the child program, so the command can fail
// before ctx is cancelled. That ordering is not expected in practice: the
// signal goroutine needs a few scheduler hand-offs, while the command first
// waits for the child to exit and be reaped. If it ever happened, invox would
// exit 1 and print the child's error instead of exiting 130.
func Main(ctx context.Context, args []string, f *Factory, commands ...*cobra.Command) int {
	cobra.MousetrapHelpText = ""
	if f.Env.Getenv("INVOX_PROMPT_DISABLED") != "" {
		f.IOStreams.SetNeverPrompt(true)
	}
	root, helpErr := newRootCmd(f, commands)
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
	var configErr *config.Error
	if configFile := f.Hosts.ConfigFile(); configFile != "" && errors.As(err, &configErr) {
		err = &configFlagError{err: err, path: configFile}
	}
	var sigErr *proc.SignalError
	if err != nil && errors.As(context.Cause(ctx), &sigErr) &&
		!(sigErr.Interrupt && errors.Is(err, cmdutil.CancelError)) {
		err = sigErr
	}
	return exitCode(f.IOStreams, f.Env.Getwd, err)
}

// warnLegacyFiles prints one line when the command read files from the
// deprecated config directory.
func warnLegacyFiles(f *Factory) {
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
