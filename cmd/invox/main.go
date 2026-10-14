package main

import (
	"os"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/app"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

func main() {
	f := cmdutil.NewFactory(iostreams.System(), run.Exec{}, env.System())
	os.Exit(app.Main(os.Args[1:], f))
}
