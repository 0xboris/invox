package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/0xboris/invox/internal/build"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

func runVersion(ios *iostreams.IOStreams, args []string) error {
	if len(args) == 1 && wantsHelp(args) {
		printVersionHelp(ios.Out)
		return nil
	}
	if len(args) > 0 {
		return cmdutil.FlagErrorf("", "unexpected arguments for version: %s", strings.Join(args, " "))
	}
	printVersion(ios.Out)
	return nil
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s version %s", commandName, strings.TrimPrefix(build.Version, "v"))
	if build.Date != "" {
		fmt.Fprintf(w, " (%s)", build.Date)
	}
	fmt.Fprintln(w)
}

func printVersionHelp(w io.Writer) {
	fmt.Fprintf(w, "Show the %s version.\n\n", commandName)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s version\n", commandName)
	fmt.Fprintf(w, "  %s --version\n", commandName)
}
