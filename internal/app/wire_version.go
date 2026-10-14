package app

import (
	"github.com/spf13/cobra"

	version "github.com/0xboris/invox/internal/features/version/ui"
	versionshared "github.com/0xboris/invox/internal/features/version/ui/shared"
)

func versionCommands(d *Dependencies) []*cobra.Command {
	f := &versionshared.Factory{IOStreams: d.IOStreams, Version: d.Version, Date: d.Date}
	return []*cobra.Command{version.NewCmdVersion(f, nil)}
}
