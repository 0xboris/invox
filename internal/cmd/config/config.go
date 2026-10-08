// Package config is the `invox config` command, which opens config.yaml in
// the editor, and its subcommands.
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/config/paths"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

type ConfigOptions struct {
	IO     *iostreams.IOStreams
	Editor *editor.Editor
	Host   func() invoice.Host
	Getwd  func() (string, error)
}

// NewCmdConfig returns the config command and its subcommands. runF replaces
// configRun in tests.
func NewCmdConfig(f *cmdutil.Factory, runF func(context.Context, *ConfigOptions) error) *cobra.Command {
	opts := &ConfigOptions{IO: f.IOStreams, Editor: f.Editor, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open config.yaml in your editor",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("config", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return configRun(cmd.Context(), opts)
		},
	}
	cmd.AddCommand(paths.NewCmdPaths(f, nil))
	return cmd
}

func configRun(ctx context.Context, opts *ConfigOptions) error {
	configPath, err := opts.Host().EditableConfigPath()
	if err != nil {
		return err
	}
	baseDir, err := opts.Getwd()
	if err != nil {
		return err
	}
	displayPath := invoice.DisplayPath(configPath, baseDir)

	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, "config", configPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
