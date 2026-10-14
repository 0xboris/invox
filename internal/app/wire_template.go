package app

import (
	"github.com/spf13/cobra"

	template "github.com/0xboris/invox/internal/features/template/ui"
	templateshared "github.com/0xboris/invox/internal/features/template/ui/shared"
)

func templateCommands(d *Dependencies) []*cobra.Command {
	f := &templateshared.Factory{IOStreams: d.IOStreams, Env: d.Env, Hosts: d.Hosts}
	return []*cobra.Command{template.NewCmdTemplate(f)}
}
