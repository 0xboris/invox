// Package edit is the `invox customer edit` command.
package edit

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/iostreams"
)

// EditOptions is what customer edit needs: its streams, the editor, the user
// directories and the parsed flags.
type EditOptions struct {
	IO      *iostreams.IOStreams
	Editor  *editor.Editor
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	CustomersPath string
}

// NewCmdEdit returns the customer edit command. runF replaces editRun in
// tests.
func NewCmdEdit(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) *cobra.Command {
	opts := &EditOptions{IO: f.IOStreams, Editor: f.Editor, Service: f.Service, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:               "edit",
		SuggestFor:        []string{"config"},
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
		Args: cmdutil.NoArgs,
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
	svc := opts.Service(cmdutil.Files{Customers: cmdutil.AbsFlag(baseDir, opts.CustomersPath)})
	customersPath, err := svc.EditablePath(billing.CustomersFile)
	if err != nil {
		return cmdutil.UsageError(err)
	}

	displayPath := cmdutil.DisplayPath(customersPath, baseDir)
	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, customersPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", customersPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
