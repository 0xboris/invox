package cmdutil

import (
	"strings"

	"github.com/spf13/cobra"
)

// NoArgs is the cobra.PositionalArgs of a command that takes no arguments.
func NoArgs(cmd *cobra.Command, args []string) error {
	return MaximumArgs(0)(cmd, args)
}

// ExactArgs is the cobra.PositionalArgs of a command that takes one argument
// for each of names. The usage error for missing arguments names them.
func ExactArgs(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < len(names) {
			return FlagErrorf("missing required arguments: %s", strings.Join(names[len(args):], " "))
		}
		return MaximumArgs(len(names))(cmd, args)
	}
}

// MaximumArgs is the cobra.PositionalArgs of a command that takes up to n
// arguments. The usage error names the arguments past n.
func MaximumArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return FlagErrorf("unexpected arguments: %s", strings.Join(args[n:], " "))
		}
		return nil
	}
}
