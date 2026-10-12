package main

import (
	"os"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func main() {
	f := factory.New(iostreams.System(), run.Exec{}, env.System())
	os.Exit(cli.Main(os.Args[1:], f))
}
