// Package edit is the `invox config edit` command, which `invox config` runs
// too.
package edit

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// EditOptions is what config edit needs: its streams, the editor and the
// user directories.
type EditOptions struct {
	IO      *iostreams.IOStreams
	Editor  *editor.Editor
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)
}

// NewCmdEdit returns the config edit command. runF replaces editRun in tests.
func NewCmdEdit(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open config.yaml in your editor",
		Long: `Open config.yaml in your editor.

Opens the resolved config.yaml, or the file given with --config. If it does
not exist yet, creates it from the template that ` + "`invox help config`" + ` shows.
`,
		Example: `$ invox config edit
$ invox --config ./config.yaml config edit
`,
	}
	Configure(cmd, f, runF)
	return cmd
}

// Configure gives cmd the arguments and run of config edit. The config
// command, which config edit is the default of, uses it too.
func Configure(cmd *cobra.Command, f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) {
	opts := &EditOptions{IO: f.IOStreams, Editor: f.Editor, Service: f.Service, Getwd: f.Env.Getwd}
	cmd.ValidArgsFunction = cobra.NoFileCompletions
	cmd.Args = cmdutil.NoArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if runF != nil {
			return runF(cmd.Context(), opts)
		}
		return editRun(cmd.Context(), opts)
	}
}

func editRun(ctx context.Context, opts *EditOptions) error {
	configPath, err := opts.Service(cmdutil.Files{}).EditablePath(billing.ConfigFile)
	if err != nil {
		return err
	}
	baseDir, err := opts.Getwd()
	if err != nil {
		return err
	}
	displayPath := cmdutil.DisplayPath(configPath, baseDir)

	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, configPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
