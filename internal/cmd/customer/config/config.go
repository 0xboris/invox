// Package config is the `invox customer config` command.
package config

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

type ConfigOptions struct {
	IO     *iostreams.IOStreams
	Editor *editor.Editor
	Host   func() invoice.Host
	Getwd  func() (string, error)

	CustomersPath string
}

// NewCmdConfig returns the customer config command. runF replaces configRun
// in tests.
func NewCmdConfig(f *cmdutil.Factory, runF func(context.Context, *ConfigOptions) error) *cobra.Command {
	opts := &ConfigOptions{IO: f.IOStreams, Editor: f.Editor, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open customers.yaml in your editor",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("customer config", "unexpected arguments: %s", strings.Join(args, " "))
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
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	return cmd
}

func configRun(ctx context.Context, opts *ConfigOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	customersPath, err := cmdutil.SupportPath(opts.Host(), "customer config", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}

	displayPath := invoice.DisplayPath(customersPath, baseDir)
	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, "customer config", customersPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", customersPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
