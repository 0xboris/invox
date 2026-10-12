// Package edit is the `invox archive edit` command.
package edit

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/iostreams"
)

// EditOptions is what archive edit needs: its streams, the user directories
// and the archived file to copy.
type EditOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	Filename string
	Force    bool
	DryRun   bool
	Exporter *cmdutil.Exporter
}

// editJSON is the --json output of archive edit: the working copy it wrote
// and the archived invoice it copied.
type editJSON struct {
	Path         string `json:"path"`
	ArchivedPath string `json:"archivedPath"`
}

// NewCmdEdit returns the archive edit command. runF replaces editRun in
// tests.
func NewCmdEdit(f *cmdutil.Factory, runF func(context.Context, *EditOptions) error) *cobra.Command {
	opts := &EditOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	if runF == nil {
		runF = editRun
	}
	cmd := &cobra.Command{
		Use:               "edit FILENAME",
		Short:             "Copy an archived invoice into the current directory and mark it as editing",
		ValidArgsFunction: cmdutil.CompleteArchivedInvoices(f),
		Long: `Copy an archived invoice into the current directory and mark it as editing.

Required inputs:
  FILENAME                  Required positional argument

Default lookup:
` +
			helptext.LookupArchive +
			`
Behavior:
  Copies the archived invoice from archive.dir into the current directory.
  The working copy is written as YAML with invoice.status set to editing.
  Re-running invox archive add on that working copy replaces the archived invoice.
  It asks first, or needs --yes without a terminal, and keeps the previous version in archive.dir/.history.
`,
		Example: `$ invox archive edit 2026-03-06.yaml
$ invox archive edit customer-a/2026-03-06.yaml
$ invox archive edit 2026-03-06.yaml --force
$ invox archive edit 2026-03-06.yaml --json path
`,
		Args: cmdutil.ExactArgs("FILENAME"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Filename = strings.TrimSpace(args[0])
			return runF(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite an existing working copy")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print where the working copy would go and write nothing")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, editJSON{})
	return cmd
}

func editRun(_ context.Context, opts *EditOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)

	edited, err := opts.Service(cmdutil.Files{}).EditArchived(opts.Filename, baseDir, billing.EditOptions{
		Overwrite: opts.Force,
		DryRun:    opts.DryRun,
	})
	outputPath, archivePath := edited.Path, edited.Archived
	if err != nil {
		return cmdutil.OutputError(err, "a different working directory")
	}

	verb := "Editing"
	if opts.DryRun {
		verb = "Would copy"
	}
	fmt.Fprintf(opts.IO.ErrOut, "%s %s -> %s\n", verb, cmdutil.DisplayPath(archivePath, baseDir), cmdutil.DisplayPath(outputPath, baseDir))
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, editJSON{Path: outputPath, ArchivedPath: archivePath})
	}
	fmt.Fprintln(opts.IO.Out, cmdutil.DisplayPath(outputPath, baseDir))
	return nil
}
