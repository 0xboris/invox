// Package list is the `invox archive list` command.
package list

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

// ListOptions is what archive list needs: its streams and the user
// directories.
type ListOptions struct {
	IO   *iostreams.IOStreams
	Host func() invoice.Host
}

// NewCmdList returns the archive list command. runF replaces listRun in
// tests.
func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, Host: f.Host}
	return &cobra.Command{
		Use:   "list",
		Short: "List archived invoices from the configured archive directory",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("archive list", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}
}

func listRun(opts *ListOptions) error {
	h := opts.Host()
	archivedInvoices, err := h.ListArchivedInvoices()
	if err != nil {
		return err
	}
	archiveDir, err := h.ResolveArchiveDir()
	if err != nil {
		return err
	}

	list := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "FILE"}, {Header: "CUSTOMER"}, {Header: "ISSUE DATE"}, {Header: "STATUS"}},
		EmptyHint: "No archived invoices found in " + archiveDir,
	}
	if archiveDir == "" {
		list.EmptyHint = "No archived invoices found"
	}
	for _, archived := range archivedInvoices {
		list.AddRow(archived.Filename, archived.CustomerID, archived.IssueDate, archived.Status)
	}
	list.Print(opts.IO)
	return nil
}
