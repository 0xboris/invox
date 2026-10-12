// Package version is the `invox version` command.
package version

import (
	"context"
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
func NewCmdVersion(f *cmdutil.Factory, runF func(context.Context, *VersionOptions) error) *cobra.Command {
	opts := &VersionOptions{IO: f.IOStreams}
	if runF == nil {
		runF = versionRun
	}
	return &cobra.Command{
		Use:               "version",
		Short:             "Show the invox version",
		ValidArgsFunction: cobra.NoFileCompletions,
		Example: `$ invox version
$ invox --version
`,
		Args: cmdutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runF(cmd.Context(), opts)
		},
	}
}

func versionRun(_ context.Context, opts *VersionOptions) error {
	w := opts.IO.Out
	fmt.Fprintf(w, "invox version %s", strings.TrimPrefix(build.Version, "v"))
	if build.Date != "" {
		fmt.Fprintf(w, " (%s)", build.Date)
	}
	fmt.Fprintln(w)
	return nil
}
