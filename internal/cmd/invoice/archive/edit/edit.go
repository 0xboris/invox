// Package edit is the `invox archive edit` command.
package edit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// EditOptions is what archive edit needs: its streams, the user directories
// and the archived file to copy.
type EditOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)

	Filename string
}

// NewCmdEdit returns the archive edit command. runF replaces editRun in
// tests.
func NewCmdEdit(f *cmdutil.Factory, runF func(*EditOptions) error) *cobra.Command {
	opts := &EditOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd}
	return &cobra.Command{
		Use:   "edit FILENAME",
		Short: "Copy an archived invoice into the current directory and mark it as editing",
		Args: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				return cmdutil.FlagErrorf("archive edit", "missing required arguments: FILENAME")
			case len(args) > 1:
				return cmdutil.FlagErrorf("archive edit", "unexpected arguments: %s", strings.Join(args[1:], " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Filename = strings.TrimSpace(args[0])
			if runF != nil {
				return runF(opts)
			}
			return editRun(opts)
		},
	}
}

func editRun(opts *EditOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)

	outputPath, archivePath, err := opts.Host().EditArchivedInvoice(opts.Filename, baseDir)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.ErrOut, "Editing %s -> %s\n", invoice.DisplayPath(archivePath, baseDir), invoice.DisplayPath(outputPath, baseDir))
	fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(outputPath, baseDir))
	return nil
}
