// Package list is the `invox archive list` command.
package list

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/shared"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

// ListOptions is what archive list needs: its streams, the working
// directory and the user directories.
type ListOptions struct {
	IO      *iostreams.IOStreams
	Getwd   func() (string, error)
	Service func(cmdutil.Files) *billing.Service

	Exporter *cmdutil.Exporter
}

// archivedJSON is an archived invoice in --json output. File is the path
// below the archive directory, as the table prints it.
type archivedJSON struct {
	File       string `json:"file"`
	Path       string `json:"path"`
	CustomerID string `json:"customerId"`
	Number     string `json:"number"`
	IssueDate  string `json:"issueDate"`
	Status     string `json:"status"`
}

// NewCmdList returns the archive list command. runF replaces listRun in
// tests.
func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, Getwd: f.Env.Getwd, Service: f.Service}
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List archived invoices from the configured archive directory",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `List archived invoices from the configured archive directory.

Default lookup:
` +
			helptext.LookupArchive +
			`
Output:
  One archived invoice per line as FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS
  On a terminal, aligned columns under a header. Piped, \, tab, CR and LF in a field are written as \\, \t, \r and \n.
`,
		Example: `$ invox archive list
$ invox archive list --json file,customerId,number
`,
		Args: cmdutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, archivedJSON{})
	return cmd
}

func listRun(opts *ListOptions) error {
	list, err := opts.Service(cmdutil.Files{}).ListArchive()
	if err != nil {
		return err
	}
	archivedInvoices, archiveDir := list.Entries, list.Dir
	// The working directory only shortens the paths of the warning.
	baseDir := ""
	if cwd, err := opts.Getwd(); err == nil {
		baseDir = filepath.Clean(cwd)
	}
	shared.WarnUnread(opts.IO, list.Unread, baseDir)

	if opts.Exporter != nil {
		items := make([]archivedJSON, 0, len(archivedInvoices))
		for _, archived := range archivedInvoices {
			items = append(items, archivedJSON{
				File:       archived.Filename,
				Path:       archived.Path,
				CustomerID: archived.CustomerID,
				Number:     archived.Number,
				IssueDate:  archived.IssueDate,
				Status:     string(archived.Status),
			})
		}
		return opts.Exporter.Write(opts.IO, items)
	}

	table := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "FILE"}, {Header: "CUSTOMER"}, {Header: "ISSUE DATE"}, {Header: "STATUS"}},
		EmptyHint: "No archived invoices found in " + archiveDir,
	}
	if archiveDir == "" {
		table.EmptyHint = "No archived invoices found"
	}
	for _, archived := range archivedInvoices {
		table.AddRow(archived.Filename, archived.CustomerID, archived.IssueDate, string(archived.Status))
	}
	table.Print(opts.IO)
	return nil
}
