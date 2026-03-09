package main

import (
	"os"

	"invox/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
