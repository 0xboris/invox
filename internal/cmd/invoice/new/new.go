// Package newcmd is the `invox new` command. The package isn't named new,
// which would hide Go's builtin in the files that import it.
package newcmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
)

type NewOptions struct {
	IO      *iostreams.IOStreams
	Editor  *editor.Editor
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	CustomerID string
	OutputPath string
	Support    cmdutil.SupportPaths
	FromLast   bool
	Edit       bool
	Force      bool
	DryRun     bool
	Exporter   *cmdutil.Exporter
}

// newJSON is the --json output of new: the invoice it created.
type newJSON struct {
	Path       string `json:"path"`
	Number     string `json:"number"`
	CustomerID string `json:"customerId"`
}

// NewCmdNew returns the new command. runF replaces newRun in tests.
func NewCmdNew(f *cmdutil.Factory, runF func(context.Context, *NewOptions) error) *cobra.Command {
	opts := &NewOptions{IO: f.IOStreams, Editor: f.Editor, Service: f.Service, Getwd: f.Env.Getwd}
	if runF == nil {
		runF = newRun
	}
	cmd := &cobra.Command{
		Use:   "new CUSTOMER_ID",
		Short: "Create a new invoice YAML file with a generated number and prefilled defaults",
		Long: `Create a new invoice YAML file with a generated number and prefilled defaults.

Required inputs:
  CUSTOMER_ID                  Required positional argument

Default output:
  <invoice.number>.yaml in the current directory

Default lookup:
` +
			helptext.LookupCustomers +
			helptext.LookupIssuer +
			helptext.LookupDefaults +
			helptext.LookupArchive,
		Example: `$ invox new CUST-001
$ invox new CUST-001 -e
$ invox new CUST-001 --from-last
$ invox new CUST-001 --dry-run
$ invox new CUST-001 --json path,number
$ invox new CUST-001 -o invoices/2026-0022.yaml --defaults invoice_defaults.yaml -c customers.yaml -u issuer.yaml
`,
		Args: cmdutil.ExactArgs("CUSTOMER_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.CustomerID = strings.TrimSpace(args[0])
			if err := shared.RequireExtension(opts.OutputPath, ".yaml"); err != nil {
				return err
			}
			return runF(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output YAML path (must end with .yaml)")
	cmdutil.AddSupportFlags(cmd, f, &opts.Support, billing.CustomersFile, billing.IssuerFile, billing.DefaultsFile)
	cmd.Flags().BoolVarP(&opts.Edit, "edit", "e", false, "Open the created invoice in your editor")
	cmd.Flags().BoolVar(&opts.FromLast, "from-last", false, "Use the latest archived invoice for this customer as the source document")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite an existing output file")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the number and path the invoice would get and write nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteCustomerIDs(f)
	_ = cmd.MarkFlagFilename("output", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, newJSON{})
	return cmd
}

func newRun(ctx context.Context, opts *NewOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	svc := opts.Service(opts.Support.Files(baseDir))
	outputPath := ""
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = cmdutil.AbsPath(baseDir, opts.OutputPath)
	}

	created, err := svc.New(billing.NewRequest{
		CustomerID: opts.CustomerID,
		WorkDir:    baseDir,
		Output:     outputPath,
		FromLast:   opts.FromLast,
		Overwrite:  opts.Force,
		DryRun:     opts.DryRun,
	})
	var exists *billing.OutputExistsError
	if errors.As(err, &exists) {
		return fmt.Errorf("%s; pass --force to replace it or choose a different -o/--output path", exists)
	}
	var isDir *billing.OutputIsDirError
	if errors.As(err, &isDir) {
		return fmt.Errorf("%s; choose a different -o/--output path", isDir)
	}
	if err != nil {
		return cmdutil.UsageError(err)
	}
	shared.WarnUnread(opts.IO, created.Unread, baseDir)
	shared.WarnSkippedArchiveFiles(opts.IO, opts.CustomerID, created.Skipped, baseDir)
	displayPath := cmdutil.DisplayPath(created.Path, baseDir)
	if opts.Edit && !opts.DryRun {
		nextStep := fmt.Sprintf("edit it and run 'invox validate -i %s'", displayPath)
		err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, created.Path, nextStep)
		var flagErr *cmdutil.FlagError
		if errors.As(err, &flagErr) {
			return &cmdutil.FlagError{Err: fmt.Errorf("created %s but %w", displayPath, flagErr.Err)}
		}
		if err != nil {
			return fmt.Errorf("created %s but failed to open it: %w", displayPath, err)
		}
	}

	verb := "Created"
	if opts.DryRun {
		verb = "Would create"
	}
	fmt.Fprintf(opts.IO.ErrOut, "%s %s for %s (%s)\n", verb, displayPath, opts.CustomerID, created.Number)
	if reason := cmdutil.WhyNoPrompt(opts.IO); opts.DryRun && opts.Edit && reason != "" {
		fmt.Fprintf(opts.IO.ErrOut, "warning: -e, --edit could not open the editor: %s\n", reason)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, newJSON{Path: created.Path, Number: created.Number, CustomerID: opts.CustomerID})
	}
	fmt.Fprintln(opts.IO.Out, displayPath)
	return nil
}
