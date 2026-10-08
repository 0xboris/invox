// Package newcmd is the `invox new` command. The package isn't named new,
// which would hide Go's builtin in the files that import it.
package newcmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

type NewOptions struct {
	IO     *iostreams.IOStreams
	Editor *editor.Editor
	Host   func() invoice.Host
	Getwd  func() (string, error)
	Now    func() time.Time

	CustomerID    string
	OutputPath    string
	DefaultsPath  string
	CustomersPath string
	IssuerPath    string
	FromLast      bool
	Edit          bool
	Force         bool
	DryRun        bool
	Exporter      *cmdutil.Exporter
}

// newJSON is the --json output of new: the invoice it created.
type newJSON struct {
	Path       string `json:"path"`
	Number     string `json:"number"`
	CustomerID string `json:"customerId"`
}

// NewCmdNew returns the new command. runF replaces newRun in tests.
func NewCmdNew(f *cmdutil.Factory, runF func(context.Context, *NewOptions) error) *cobra.Command {
	opts := &NewOptions{IO: f.IOStreams, Editor: f.Editor, Host: f.Host, Getwd: f.Env.Getwd, Now: f.Env.Now}
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
		Args: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				return cmdutil.FlagErrorf("new", "missing required arguments: CUSTOMER_ID")
			case len(args) > 1:
				return cmdutil.FlagErrorf("new", "unexpected arguments: %s", strings.Join(args[1:], " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("source") {
				cmdutil.WarnDeprecated(opts.IO.ErrOut, "-s, --source", "--defaults")
			}
			opts.CustomerID = strings.TrimSpace(args[0])
			if err := shared.RequireExtension("new", opts.OutputPath, ".yaml"); err != nil {
				return err
			}
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return newRun(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.OutputPath, "output", "o", "", "Output YAML path (must end with .yaml)")
	cmd.Flags().StringVar(&opts.DefaultsPath, "defaults", "", "Path to invoice_defaults.yaml")
	cmd.Flags().StringVarP(&opts.DefaultsPath, "source", "s", "", "Path to invoice_defaults.yaml (deprecated: use --defaults)")
	cmdutil.DeprecateFlag(cmd.Flags(), "source")
	cmd.Flags().StringVarP(&opts.CustomersPath, "customers", "c", "", "Path to customers.yaml")
	cmd.Flags().StringVarP(&opts.IssuerPath, "issuer", "u", "", "Path to issuer.yaml")
	cmd.Flags().BoolVarP(&opts.Edit, "edit", "e", false, "Open the created invoice in your editor")
	cmd.Flags().BoolVar(&opts.FromLast, "from-last", false, "Use the latest archived invoice for this customer as the source document")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite an existing output file")
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the number and path the invoice would get and write nothing")
	cmd.ValidArgsFunction = cmdutil.CompleteCustomerIDs(f)
	_ = cmd.MarkFlagFilename("output", "yaml", "yml")
	_ = cmd.MarkFlagFilename("defaults", "yaml", "yml")
	_ = cmd.MarkFlagFilename("customers", "yaml", "yml")
	_ = cmd.MarkFlagFilename("issuer", "yaml", "yml")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, newJSON{})
	return cmd
}

func newRun(ctx context.Context, opts *NewOptions) error {
	cwd, err := opts.Getwd()
	if err != nil {
		return err
	}
	baseDir := filepath.Clean(cwd)
	h := opts.Host()
	customersPath, err := cmdutil.SupportPath(h, "new", invoice.Customers, opts.CustomersPath, baseDir)
	if err != nil {
		return err
	}
	issuerPath, err := cmdutil.SupportPath(h, "new", invoice.Issuer, opts.IssuerPath, baseDir)
	if err != nil {
		return err
	}
	defaultsPath, err := cmdutil.SupportPath(h, "new", invoice.Defaults, opts.DefaultsPath, baseDir)
	var notFound *cmdutil.FlagError
	if opts.FromLast && errors.As(err, &notFound) {
		// --from-last copies the last archived invoice instead.
		defaultsPath, err = "", nil
	}
	if err != nil {
		return err
	}
	outputPath := ""
	if strings.TrimSpace(opts.OutputPath) != "" {
		outputPath = invoice.AbsPath(baseDir, opts.OutputPath)
	}

	created, err := h.CreateNewInvoice(invoice.NewInvoiceParams{
		Now:           opts.Now(),
		WorkDir:       baseDir,
		DefaultsPath:  defaultsPath,
		OutputPath:    outputPath,
		CustomersPath: customersPath,
		IssuerPath:    issuerPath,
		CustomerID:    opts.CustomerID,
		FromLast:      opts.FromLast,
		Overwrite:     opts.Force,
		DryRun:        opts.DryRun,
	})
	var exists *invoice.OutputExistsError
	if errors.As(err, &exists) {
		return fmt.Errorf("%s; pass --force to replace it or choose a different -o/--output path", exists)
	}
	if err != nil {
		return err
	}
	shared.WarnSkippedArchiveFiles(opts.IO, opts.CustomerID, created.SkippedArchiveFiles, baseDir)
	displayPath := invoice.DisplayPath(created.Path, baseDir)
	if opts.Edit && !opts.DryRun {
		nextStep := fmt.Sprintf("edit it and run 'invox validate -i %s'", displayPath)
		err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, "new", created.Path, nextStep)
		var flagErr *cmdutil.FlagError
		if errors.As(err, &flagErr) {
			return &cmdutil.FlagError{Command: flagErr.Command, Err: fmt.Errorf("created %s but %w", displayPath, flagErr.Err)}
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
