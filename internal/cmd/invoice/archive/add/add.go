// Package add is the `invox archive add` command, which archives an invoice.
package add

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// AddOptions is what archive add needs: its streams, the user directories,
// the clock and the parsed flags. Command names the command in its messages.
type AddOptions struct {
	IO    *iostreams.IOStreams
	Host  func() invoice.Host
	Getwd func() (string, error)
	Now   func() time.Time

	Command     string
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
	}
	Configure(cmd, f, runF, false)
	return cmd
}

// Configure gives cmd the arguments, flags and run of archive add. With
// deprecated set, cmd is `archive FILE`, the deprecated form of archive add:
// help and completion leave out its flags and files, and it warns before it
// archives.
func Configure(cmd *cobra.Command, f *cmdutil.Factory, runF func(context.Context, *AddOptions) error, deprecated bool) {
	opts := &AddOptions{IO: f.IOStreams, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		return shared.TakeInput(cmdutil.CommandPath(cmd), &opts.InvoicePath, args)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.Command = cmdutil.CommandPath(cmd)
		if err := shared.RequireInput(opts.Command, opts.InvoicePath); err != nil {
			return err
		}
		if deprecated {
			cmdutil.WarnDeprecated(opts.IO.ErrOut, "archive FILE", "archive add FILE")
		}
		if runF != nil {
			return runF(cmd.Context(), opts)
		}
		return addRun(cmd.Context(), opts)
	}
	cmd.Flags().StringVarP(&opts.InvoicePath, "input", "i", "", "Input invoice YAML file")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Replace an archived invoice without asking")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print where the invoice would be archived and change nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteInputFile("yaml", "yml")
	_ = cmd.MarkFlagFilename("input", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, addJSON{})
	if deprecated {
		cmd.ValidArgsFunction = cobra.NoFileCompletions
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { flag.Hidden = true })
		delete(cmd.Annotations, cmdutil.JSONFieldsAnnotation)
	}
}

func addRun(ctx context.Context, opts *AddOptions) error {
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
		result, err = shared.ArchiveWithConfirmation(ctx, opts.IO, opts.Host(), opts.Now, opts.Command, invoicePath, baseDir, opts.Yes, "")
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
		return opts.Exporter.Write(opts.IO, addJSON{Path: result.Path, Input: invoicePath, Replaced: replaced})
	}
	fmt.Fprintln(opts.IO.Out, invoice.DisplayPath(result.Path, baseDir))
	return nil
}
