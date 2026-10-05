package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0xboris/invox/internal/build"
)

func runVersion(args []string) int {
	if len(args) == 1 && wantsHelp(args) {
		printVersionHelp(os.Stdout)
		return 0
	}
	if len(args) > 0 {
		return rootUsageError(fmt.Sprintf("unexpected arguments for version: %s", strings.Join(args, " ")))
	}
	printVersion(os.Stdout)
	return 0
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
