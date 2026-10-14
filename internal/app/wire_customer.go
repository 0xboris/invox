package app

import (
	"github.com/spf13/cobra"

	customer "github.com/0xboris/invox/internal/features/customer/ui"
	customershared "github.com/0xboris/invox/internal/features/customer/ui/shared"
)

func customerCommands(d *Dependencies) []*cobra.Command {
	f := &customershared.Factory{IOStreams: d.IOStreams, Env: d.Env, Hosts: d.Hosts, Editor: d.Editor}
	return []*cobra.Command{customer.NewCmdCustomer(f)}
}
