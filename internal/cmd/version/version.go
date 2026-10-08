// Package version is the `invox version` command.
package version

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/build"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

type VersionOptions struct {
	IO *iostreams.IOStreams
}

// NewCmdVersion returns the version command. runF replaces versionRun in tests.
func NewCmdVersion(f *cmdutil.Factory, runF func(*VersionOptions) error) *cobra.Command {
	opts := &VersionOptions{IO: f.IOStreams}
	return &cobra.Command{
		Use:   "version",
		Short: "Show the invox version",
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
	fmt.Fprintf(w, "invox version %s", strings.TrimPrefix(build.Version, "v"))
	if build.Date != "" {
		fmt.Fprintf(w, " (%s)", build.Date)
	}
	fmt.Fprintln(w)
	return nil
}
