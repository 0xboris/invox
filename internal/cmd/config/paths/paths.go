// Package paths is the `invox config paths` command.
package paths

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

type PathsOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)
}

// NewCmdPaths returns the config paths command. runF replaces pathsRun in
// tests.
func NewCmdPaths(f *cmdutil.Factory, runF func(*PathsOptions) error) *cobra.Command {
	opts := &PathsOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	return &cobra.Command{
		Use:   "paths",
		Short: "Show where each config and support file is read from",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("config paths", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return pathsRun(opts)
		},
	}
}

func pathsRun(opts *PathsOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	reports, err := opts.Host().Paths(cwd)
	if err != nil {
		return err
	}
	t := tableprinter.Table{Columns: []tableprinter.Column{{Header: "NAME"}, {Header: "PATH"}, {Header: "SOURCE"}}}
	for _, r := range reports {
		t.AddRow(r.Name, r.Path, sourceWords[r.Source])
	}
	t.Print(opts.IO)
	return nil
}

// sourceWords are the SOURCE column.
var sourceWords = map[invoice.Source]string{
	invoice.SourceNone:     "none",
	invoice.SourceExplicit: "flag",
	invoice.SourceEnvDir:   "env",
	invoice.SourceDefault:  "default",
	invoice.SourceLegacy:   "legacy",
	invoice.SourceProject:  "project",
	invoice.SourceConfig:   "config",
}
