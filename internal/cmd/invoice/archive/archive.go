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
	"github.com/0xboris/invox/internal/cli/helptext"
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
	DryRun      bool
	Exporter    *cmdutil.Exporter
}

// archiveJSON is the --json output of archive: where the invoice was
// archived and the archived files it replaced.
type archiveJSON struct {
	Path     string         `json:"path"`
	Input    string         `json:"input"`
	Replaced []replacedJSON `json:"replaced"`
}

// replacedJSON is an archived file that archive replaced, and where its
// previous version is kept.
type replacedJSON struct {
	Path       string `json:"path"`
	BackupPath string `json:"backupPath"`
}

// NewCmdArchive returns the archive command and its subcommands. runF
// replaces archiveRun in tests.
func NewCmdArchive(f *cmdutil.Factory, runF func(context.Context, *ArchiveOptions) error) *cobra.Command {
	opts := &ArchiveOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
	cmd := &cobra.Command{
		Use:   "archive [INVOICE.yaml]",
		Short: "Archive a built or edited invoice YAML file into the configured archive directory",
		Long: `Archive a built or edited invoice YAML file into the configured archive directory.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupArchive +
			"\n" +
			helptext.ReplacingArchived(false),
		Example: `$ invox archive invoice.yaml
$ invox archive invoices/2026-0021.yaml
$ invox archive 2026-03-06.yaml --yes
$ invox archive invoice.yaml --dry-run
$ invox archive invoice.yaml --json path
$ invox archive edit 2026-03-06.yaml
$ invox archive list
`,
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
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print where the invoice would be archived and change nothing")
	cmd.AddCommand(edit.NewCmdEdit(f, nil), list.NewCmdList(f, nil))
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, archiveJSON{})
	return cmd
}

func archiveRun(ctx context.Context, opts *ArchiveOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	invoicePath := invoice.AbsPath(baseDir, opts.InvoicePath)

	var result invoice.ArchiveResult
	if opts.DryRun {
		result, err = opts.Host().ArchiveInvoice(opts.Now(), invoicePath, invoice.ArchiveOptions{Replace: true, DryRun: true})
		if err != nil {
			return err
		}
		shared.PrintArchivePreview(opts.IO, result, invoicePath, baseDir)
	} else {
		result, err = shared.ArchiveWithConfirmation(ctx, opts.IO, opts.Host(), opts.Now, "archive", invoicePath, baseDir, opts.Yes, "")
		if err != nil {
			return err
		}
		shared.PrintArchiveReplacements(opts.IO, result, baseDir)
		fmt.Fprintf(opts.IO.ErrOut, "Archived %s -> %s\n", invoice.DisplayPath(invoicePath, baseDir), invoice.DisplayPath(result.Path, baseDir))
	}
	if opts.Exporter != nil {
		replaced := make([]replacedJSON, 0, len(result.Replaced))
		for _, backup := range result.Replaced {
			replaced = append(replaced, replacedJSON{Path: backup.Path, BackupPath: backup.BackupPath})
		}
		return opts.Exporter.Write(opts.IO, archiveJSON{Path: result.Path, Input: invoicePath, Replaced: replaced})
	}
	fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(result.Path, baseDir))
	return nil
}
