// Package add is the `invox archive add` command, which archives an invoice.
package add

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
)

// AddOptions is what archive add needs: its streams, the user directories,
// the clock and the parsed flags.
type AddOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	InvoicePath string
	Yes         bool
	DryRun      bool
	Exporter    *cmdutil.Exporter
}

// addJSON is the --json output of archive add: where the invoice was
// archived and the archived files it replaced.
type addJSON struct {
	Path     string         `json:"path"`
	Input    string         `json:"input"`
	Replaced []replacedJSON `json:"replaced"`
}

// replacedJSON is an archived file that archive add replaced, and where its
// previous version is kept.
type replacedJSON struct {
	Path       string `json:"path"`
	BackupPath string `json:"backupPath"`
}

// NewCmdAdd returns the archive add command. runF replaces addRun in tests.
func NewCmdAdd(f *cmdutil.Factory, runF func(context.Context, *AddOptions) error) *cobra.Command {
	opts := &AddOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	if runF == nil {
		runF = addRun
	}
	cmd := &cobra.Command{
		Use:   "add [INVOICE.yaml]",
		Short: "Archive a built or edited invoice YAML file into the configured archive directory",
		Long: `Archive a built or edited invoice YAML file into the configured archive directory.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default lookup:
` +
			helptext.LookupArchive +
			"\n" +
			helptext.ReplacingArchived(false),
		Example: `$ invox archive add invoice.yaml
$ invox archive add invoices/2026-0021.yaml
$ invox archive add 2026-03-06.yaml --yes
$ invox archive add invoice.yaml --dry-run
$ invox archive add invoice.yaml --json path
`,
		Args: shared.TakeInput(opts.Getwd, &opts.InvoicePath),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireInput(opts.InvoicePath); err != nil {
				return err
			}
			return runF(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Replace an archived invoice without asking")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print where the invoice would be archived and change nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, addJSON{})
	return cmd
}

func addRun(ctx context.Context, opts *AddOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	invoicePath := cmdutil.AbsPath(baseDir, opts.InvoicePath)
	svc := opts.Service(cmdutil.Files{})

	var result billing.ArchiveResult
	if opts.DryRun {
		result, err = svc.Archive(invoicePath, billing.ArchiveOptions{Replace: true, DryRun: true})
		if err != nil {
			return err
		}
		shared.WarnUnread(opts.IO, result.Unread, baseDir)
		shared.PrintArchivePreview(opts.IO, result, invoicePath, baseDir)
	} else {
		result, err = svc.Archive(invoicePath, billing.ArchiveOptions{Replace: opts.Yes, Confirm: shared.ConfirmReplace(ctx, opts.IO, invoicePath, baseDir, "")})
		if err != nil {
			return err
		}
		shared.WarnUnread(opts.IO, result.Unread, baseDir)
		shared.PrintArchiveReplacements(opts.IO, result, baseDir)
		fmt.Fprintf(opts.IO.ErrOut, "Archived %s -> %s\n", cmdutil.DisplayPath(invoicePath, baseDir), cmdutil.DisplayPath(result.Path, baseDir))
	}
	if opts.Exporter != nil {
		replaced := make([]replacedJSON, 0, len(result.Replaced))
		for _, backup := range result.Replaced {
			replaced = append(replaced, replacedJSON{Path: backup.Path, BackupPath: backup.BackupPath})
		}
		return opts.Exporter.Write(opts.IO, addJSON{Path: result.Path, Input: invoicePath, Replaced: replaced})
	}
	fmt.Fprintln(opts.IO.Out, cmdutil.DisplayPath(result.Path, baseDir))
	return nil
}
