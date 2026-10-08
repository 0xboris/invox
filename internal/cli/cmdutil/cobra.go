package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagErrorFunc turns a cobra flag-parsing error into a *FlagError for cmd.
// An unknown long flag gets the closest flag name as a suggestion.
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if name, ok := strings.CutPrefix(err.Error(), "unknown flag: --"); ok {
		if suggestion := closestFlag(cmd, name); suggestion != "" {
			err = fmt.Errorf("%w; did you mean --%s?", err, suggestion)
		}
	}
	return &FlagError{Command: CommandPath(cmd), Err: err}
}

// UnknownSubcommandError is the usage error for an unknown subcommand of
// cmd, with the closest subcommands as a suggestion.
func UnknownSubcommandError(cmd *cobra.Command, name string) error {
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	message := fmt.Sprintf("unknown %s subcommand %q", CommandPath(cmd), name)
	if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
		message += fmt.Sprintf("; did you mean %q?", strings.Join(suggestions, `" or "`))
	}
	return FlagErrorf(CommandPath(cmd), "%s", message)
}

// CommandPath is cmd's path without the root command, for example
// "template list", or "" for the root.
func CommandPath(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return ""
	}
	return strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
}

// closestFlag returns the name of cmd's flag nearest to name, or "" when none
// is within two edits.
func closestFlag(cmd *cobra.Command, name string) string {
	best, bestDistance := "", 3
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if d := editDistance(name, f.Name); d < bestDistance {
			best, bestDistance = f.Name, d
		}
	})
	return best
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
