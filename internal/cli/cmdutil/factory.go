package cmdutil

import (
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

// Files are the files a command names on its command line: the support
// files as absolute paths, "" to look them up, and whether -o names the
// email draft.
type Files struct {
	Customers string
	Issuer    string
	Defaults  string
	// EmailOutput is set when the email draft goes to a file the user
	// named, which rules out the mail app.
	EmailOutput bool
}

// Factory holds what commands use to reach outside invox: the standard
// streams, the process environment, the programs the user interacts with
// and the use cases.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Editor    *editor.Editor
	Opener    *opener.Opener
	// ConfigFile is the --config value as typed, "" when not given. Main
	// sets it before anything calls Service.
	ConfigFile string
	// Service returns the use cases for a command that names files.
	Service func(Files) *billing.Service
	// Locations are where invox keeps its files by default, for help
	// texts.
	Locations func() helptext.Locations
}
