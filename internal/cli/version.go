package cli

import (
	"fmt"
	"io"
)

// Version defaults to a development marker and can be overridden at build time
// with -ldflags="-X invox/internal/cli.Version=<value>".
var Version = "dev"

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n", commandName, Version)
}
