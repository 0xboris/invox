// Package initcmd is the `invox init` command. The package isn't named init,
// which Go reserves for package initializers.
package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// InitOptions is what init needs: its streams, the user directories and
// the parsed flags.
type InitOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
}

// NewCmdInit returns the init command. runF replaces initRun in tests.
func NewCmdInit(f *cmdutil.Factory, runF func(context.Context, *InitOptions) error) *cobra.Command {
	opts := &InitOptions{IO: f.IOStreams, Service: f.Service}
	cmd := &cobra.Command{
		Use:               "init",
		Short:             "Create starter support files in the global config directory",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `Create starter support files in the global config directory.

Behavior:
  Creates the global config directory if it does not exist yet.
  Writes starter versions of config.yaml, customers.yaml, issuer.yaml,
  invoice_defaults.yaml, and template.tex.
  Existing non-empty files are left unchanged.

Config directory:
  {{.ConfigDir}}
`,
		Example: `$ invox init
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return initRun(cmd.Context(), opts)
		},
	}
	return cmd
}

func initRun(ctx context.Context, opts *InitOptions) error {
	initialized, err := opts.Service(cmdutil.Files{}).Init()
	if err != nil {
		return err
	}
	configDir := initialized.ConfigDir

	fmt.Fprintf(opts.IO.ErrOut, "Initialized %s\n", configDir)
	for _, result := range initialized.Files {
		status := "exists"
		if result.Created {
			status = "created"
		}
		fmt.Fprintf(opts.IO.ErrOut, "%s %s\n", status, cmdutil.DisplayPath(result.Path, configDir))
	}
	return nil
}
