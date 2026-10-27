package app

import (
	"github.com/spf13/cobra"

	config "github.com/0xboris/invox/internal/features/config/ui"
	configshared "github.com/0xboris/invox/internal/features/config/ui/shared"
	configusecases "github.com/0xboris/invox/internal/features/config/usecases"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

func configCommands(d *Dependencies) []*cobra.Command {
	f := &configshared.Factory{IOStreams: d.IOStreams, Env: d.Env, Hosts: d.Hosts, EditConfig: configusecases.NewEditConfig(d.Editor)}
	return []*cobra.Command{config.NewCmdConfig(f, nil)}
}

func configTopics() []cmdutil.Topic {
	return config.Topics()
}
