// Package archive is the `invox archive` noun: its add, edit and list
// subcommands.
package archive

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/add"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/edit"
	"github.com/0xboris/invox/internal/cmd/invoice/archive/list"
)

// NewCmdArchive returns the archive command and its subcommands. Without a
// subcommand it prints its help.
func NewCmdArchive(f *cmdutil.Factory) *cobra.Command {
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
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return cmdutil.UnknownSubcommandError(cmd, args[0])
		},
	}
	cmd.AddCommand(add.NewCmdAdd(f, nil), edit.NewCmdEdit(f, nil), list.NewCmdList(f, nil))
	return cmd
}
