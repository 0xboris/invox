package app

import cli "github.com/0xboris/invox/internal/features/cli/ui"

func cliFactory(d *Dependencies) *cli.Factory {
	return &cli.Factory{IOStreams: d.IOStreams, Env: d.Env, Hosts: d.Hosts, Topics: topics(d)}
}
