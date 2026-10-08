// Package template is the `invox template` noun.
package template

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/template/list"
)

// NewCmdTemplate returns the template command and its subcommands. Without a
// subcommand it prints its help.
func NewCmdTemplate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template <subcommand>",
		Short: "Template-related commands",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return cmdutil.UnknownSubcommandError(cmd, args[0])
		},
	}
	cmd.AddCommand(list.NewCmdList(f, nil))
	return cmd
}
