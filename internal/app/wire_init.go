package app

import (
	"github.com/spf13/cobra"

	initcmd "github.com/0xboris/invox/internal/features/init/ui"
	initshared "github.com/0xboris/invox/internal/features/init/ui/shared"
)

func initCommands(d *Dependencies) []*cobra.Command {
	f := &initshared.Factory{IOStreams: d.IOStreams, Hosts: d.Hosts}
	return []*cobra.Command{initcmd.NewCmdInit(f, nil)}
}
