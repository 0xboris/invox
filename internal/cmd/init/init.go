// Package initcmd is the `invox init` command. The package isn't named init,
// which Go reserves for package initializers.
package initcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// InitOptions is what init needs: its streams, the user directories and
// the parsed flags.
type InitOptions struct {
	IO   *iostreams.IOStreams
	Host func() invoice.Host

	Force bool
}

// NewCmdInit returns the init command. runF replaces initRun in tests.
func NewCmdInit(f *cmdutil.Factory, runF func(context.Context, *InitOptions) error) *cobra.Command {
	opts := &InitOptions{IO: f.IOStreams, Host: f.Host}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create starter support files in the global config directory",
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
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Copy files from the deprecated config directory without asking")
	return cmd
}

func initRun(ctx context.Context, opts *InitOptions) error {
	h := opts.Host()
	if err := copyLegacyFiles(ctx, opts.IO, h, opts.Force); err != nil {
		return err
	}

	configDir, results, err := h.InitializeConfigDir()
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.ErrOut, "Initialized %s\n", configDir)
	for _, result := range results {
		status := "exists"
		if result.Created {
			status = "created"
		}
		fmt.Fprintf(opts.IO.ErrOut, "%s %s\n", status, invoice.DisplayPath(result.Path, configDir))
	}
	return nil
}

// copyLegacyFiles copies the files of the deprecated config directory that
// the config directory lacks, after asking, unless force is set.
func copyLegacyFiles(ctx context.Context, ios *iostreams.IOStreams, h invoice.Host, force bool) error {
	missing, err := h.LegacyFilesToCopy()
	if err != nil || len(missing) == 0 {
		return err
	}
	legacyDir, configDir := h.LegacyConfigDir(), h.ConfigDir()
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

	copied, err := h.CopyLegacyFiles()
	for _, rel := range copied {
		fmt.Fprintf(ios.ErrOut, "copied %s from %s\n", rel, legacyDir)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.ErrOut, "invox no longer reads %s for these files; remove it once you are happy with %s\n", legacyDir, configDir)
	return nil
}
