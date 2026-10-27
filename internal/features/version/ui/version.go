// Package version is the `invox version` command.
package version

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/features/version/ui/shared"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

type VersionOptions struct {
	IO *iostreams.IOStreams
	// Version is the release version, such as "v1.2.3", or "DEV", and Date
	// its release date (YYYY-MM-DD) or "".
	Version, Date string
}

// NewCmdVersion returns the version command. runF replaces versionRun in tests.
func NewCmdVersion(f *shared.Factory, runF func(*VersionOptions) error) *cobra.Command {
	opts := &VersionOptions{IO: f.IOStreams, Version: f.Version, Date: f.Date}
	return &cobra.Command{
		Use:               "version",
		Short:             "Show the invox version",
		ValidArgsFunction: cobra.NoFileCompletions,
		Example: `$ invox version
$ invox --version
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("version", "unexpected arguments for version: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return versionRun(opts)
		},
	}
}

func versionRun(opts *VersionOptions) error {
	w := opts.IO.Out
	fmt.Fprintf(w, "invox version %s", strings.TrimPrefix(opts.Version, "v"))
	if opts.Date != "" {
		fmt.Fprintf(w, " (%s)", opts.Date)
	}
	fmt.Fprintln(w)
	return nil
}
