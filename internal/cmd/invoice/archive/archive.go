// Package archive is the `invox archive` noun: its add, edit and list
// subcommands, and `archive FILE`, the deprecated form of `archive add FILE`.
package archive

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/add"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/edit"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/list"
)

// NewCmdArchive returns the archive command and its subcommands. runF
// replaces the run of `archive FILE` in tests.
func NewCmdArchive(f *cmdutil.Factory, runF func(context.Context, *add.AddOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <subcommand>",
		Short: "Archive invoices, and list or edit archived ones",
		Long: `Archive invoices, and list or edit archived ones.

Default lookup:
` + helptext.LookupArchive,
		Example: `$ invox archive add invoice.yaml
$ invox archive list
$ invox archive edit 2026-03-06.yaml
`,
	}
	add.Configure(cmd, f, runF, true)
	cmd.AddCommand(add.NewCmdAdd(f, nil), edit.NewCmdEdit(f, nil), list.NewCmdList(f, nil))
	return cmd
}
