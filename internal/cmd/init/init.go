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

	Force bool
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
  When the deprecated invoice-tool directory has files the config directory
  lacks, asks first, then copies them in before writing the starter files.
  Nothing is replaced, and the invoice-tool directory is left in place.

Config directory:
  {{.ConfigDir}}
`,
		Example: `$ invox init
$ invox init --force
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("init", "unexpected arguments: %s", strings.Join(args, " "))
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
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Copy files from the deprecated config directory without asking (required without a terminal)")
	return cmd
}

func initRun(ctx context.Context, opts *InitOptions) error {
	svc := opts.Service(cmdutil.Files{})
	if err := copyLegacyFiles(ctx, opts.IO, svc, opts.Force); err != nil {
		return err
	}

	initialized, err := svc.Init()
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

// copyLegacyFiles copies the files of the deprecated config directory that
// the config directory lacks, after asking, unless force is set.
func copyLegacyFiles(ctx context.Context, ios *iostreams.IOStreams, svc *billing.Service, force bool) error {
	missing, err := svc.LegacyFiles()
	if err != nil || len(missing) == 0 {
		return err
	}
	locations := svc.Locations()
	legacyDir, configDir := locations.LegacyDir, locations.ConfigDir
	if !force {
		if !ios.CanPrompt() {
			return cmdutil.FlagErrorf("init", "the deprecated config directory %s has files that %s lacks; pass --force to copy them (no terminal to ask on)", legacyDir, configDir)
		}
		confirmed, err := cmdutil.Confirm(ctx, ios, fmt.Sprintf("Copy %s from %s to %s?", strings.Join(missing, ", "), legacyDir, configDir))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(ios.ErrOut, "not initialized; nothing was changed\n")
			return cmdutil.CancelError
		}
	}

	copied, err := svc.CopyLegacyFiles()
	for _, rel := range copied {
		fmt.Fprintf(ios.ErrOut, "copied %s from %s\n", rel, legacyDir)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.ErrOut, "invox no longer reads %s for these files; remove it once you are happy with %s\n", legacyDir, configDir)
	return nil
}
