package main

import (
	"os"

	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/iostreams"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], iostreams.System()))
}
