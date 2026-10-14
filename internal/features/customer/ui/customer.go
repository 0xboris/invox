// Package customer is the `invox customer` noun.
package customer

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/domain/helptext"
	"github.com/0xboris/invox/internal/features/customer/ui/edit"
	"github.com/0xboris/invox/internal/features/customer/ui/list"
	"github.com/0xboris/invox/internal/features/customer/ui/shared"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

// NewCmdCustomer returns the customer command and its subcommands. Without a
// subcommand it prints its help.
func NewCmdCustomer(f *shared.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customer <subcommand>",
		Short: "Customer-related commands",
		Long: `Customer-related commands.

Default lookup:
  customers.yaml: upward project search, then {{.GlobalCustomersPath}}

Documentation:
  Run ` + "`" + `invox help customers` + "`" + ` for the customers.yaml schema reference.

` +
			helptext.CustomerFieldReference() +
			"\n" +
			helptext.CustomerYAMLExample(),
		Example: `$ invox customer list
$ invox customer list -c customers.yaml
$ invox customer edit
`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return cmdutil.UnknownSubcommandError(cmd, args[0])
		},
	}
	cmd.AddCommand(
		list.NewCmdList(f, nil),
		edit.NewCmdEdit(f, nil),
		edit.NewCmdConfig(f, nil),
	)
	return cmd
}
