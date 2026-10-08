// Package archive is the `invox archive` command, which archives an invoice,
// and its edit and list subcommands.
package archive

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/edit"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/list"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// ArchiveOptions is what archive needs: its streams, the user directories,
// the clock and the parsed flags.
type ArchiveOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)
	Now   func() time.Time

	InvoicePath string
	Yes         bool
}

// NewCmdArchive returns the archive command and its subcommands. runF
// replaces archiveRun in tests.
func NewCmdArchive(f *cmdutil.Factory, runF func(context.Context, *ArchiveOptions) error) *cobra.Command {
	opts := &ArchiveOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
	cmd := &cobra.Command{
		Use:   "archive [INVOICE.yaml]",
		Short: "Archive a built or edited invoice YAML file into the configured archive directory",
		Args: func(cmd *cobra.Command, args []string) error {
			if rest := shared.TakeInput(&opts.InvoicePath, args); len(rest) > 0 {
				return cmdutil.FlagErrorf("archive", "unexpected arguments: %s", strings.Join(rest, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.InvoicePath) == "" {
				return cmdutil.FlagErrorf("archive", "missing required input: INVOICE.yaml or -i, --input")
			}
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return archiveRun(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Replace an archived invoice without asking")
	cmd.AddCommand(edit.NewCmdEdit(f, nil), list.NewCmdList(f, nil))
	return cmd
}

func archiveRun(ctx context.Context, opts *ArchiveOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)

	result, err := shared.ArchiveWithConfirmation(ctx, opts.IO, opts.Host(), opts.Now, "archive", invoicePath, baseDir, opts.Yes, "")
	if err != nil {
		return err
	}

	shared.PrintArchiveReplacements(opts.IO, result, baseDir)
	fmt.Fprintf(opts.IO.ErrOut, "Archived %s -> %s\n", invoice.DisplayPath(invoicePath, baseDir), invoice.DisplayPath(result.Path, baseDir))
	fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(result.Path, baseDir))
	return nil
}
