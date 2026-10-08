// Package edit is the `invox customer edit` command, also run as the
// deprecated `invox customer config`.
package edit

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/store"
)

// EditOptions is what customer edit needs: its streams, the editor, the user
// directories and the parsed flags. Command names the command in its
// messages.
type EditOptions struct {
	IO     *iostreams.IOStreams
	Editor *editor.Editor
	Host   func() store.Host
	Getwd  func() (string, error)

	Command       string
	CustomersPath string
}

// NewCmdEdit returns the customer edit command. runF replaces editRun in
// tests.
func NewCmdEdit(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) *cobra.Command {
	return newCmd(f, runF, "edit")
}

// NewCmdConfig returns `customer config`, the hidden, deprecated name of
// customer edit, which warns before it runs.
func NewCmdConfig(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) *cobra.Command {
	cmd := newCmd(f, runF, "config")
	cmd.Hidden = true
	run := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmdutil.WarnDeprecated(cmd.ErrOrStderr(), "customer config", "customer edit")
		return run(cmd, args)
	}
	return cmd
}

func newCmd(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error, name string) *cobra.Command {
	opts := &EditOptions{IO: f.IOStreams, Editor: f.Editor, Host: f.Host, Getwd: f.Env.Getwd, Command: "customer " + name}
	cmd := &cobra.Command{
		Use:               name,
		Short:             "Open customers.yaml in your editor",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `Open customers.yaml in your editor.

Default lookup:
` +
			helptext.LookupCustomers +
			"\n" +
			helptext.CustomerFieldReference() +
			"\n" +
			helptext.CustomerYAMLExample(),
		Example: `$ invox customer edit
$ invox customer edit -c customers.yaml
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf(opts.Command, "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return editRun(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	return cmd
}

func editRun(ctx context.Context, opts *EditOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	customersPath, err := cmdutil.SupportPath(opts.Host(), opts.Command, store.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}

	displayPath := store.DisplayPath(customersPath, baseDir)
	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, opts.Command, customersPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", customersPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
