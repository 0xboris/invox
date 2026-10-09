// Package paths is the `invox config paths` command.
package paths

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

type PathsOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
}

// NewCmdPaths returns the config paths command. runF replaces pathsRun in
// tests.
func NewCmdPaths(f *cmdutil.Factory, runF func(*PathsOptions) error) *cobra.Command {
	opts := &PathsOptions{IO: f.IOStreams, Service: f.Service}
	return &cobra.Command{
		Use:               "paths",
		Short:             "Show where each config and support file is read from",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `Show where each config and support file is read from.

Output:
  One row per path with the columns NAME, PATH and SOURCE, in this order:
  config-dir, config, customers, issuer, defaults, template, archive.
  The support files are looked up from the current directory, as a command
  run here would. PATH is empty when nothing is found. On a terminal the
  rows are aligned under a header; piped, they are tab-separated.

Sources:
  flag     the --config option
  env      INVOX_CONFIG_DIR
  default  the default config or archive directory
  project  the upward search from the current directory
  config   a paths.* or archive.dir setting in config.yaml
  none     not found
`,
		Example: `$ invox config paths
`,
		Args: cmdutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return pathsRun(opts)
		},
	}
}

func pathsRun(opts *PathsOptions) error {
	reports, err := opts.Service(cmdutil.Files{}).Paths()
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
var sourceWords = map[billing.Source]string{
	billing.SourceNone:     "none",
	billing.SourceExplicit: "flag",
	billing.SourceEnvDir:   "env",
	billing.SourceDefault:  "default",
	billing.SourceProject:  "project",
	billing.SourceConfig:   "config",
}
