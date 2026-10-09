package cmdutil

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagErrorFunc turns a cobra flag-parsing error on cmd into a *FlagError.
// --json without a value is the exception: it lists the fields and exits 1.
// An unknown flag gets the closest long flag name as a suggestion. An error
// in a global flag points to the root help, which documents it.
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if jsonErr := jsonFlagWithoutValue(cmd, err); jsonErr != nil {
		return jsonErr
	}
	if name, ok := strings.CutPrefix(err.Error(), "unknown flag: -"); ok {
		if suggestion := closestFlag(cmd, strings.TrimPrefix(name, "-")); suggestion != "" {
			err = fmt.Errorf("%w; did you mean --%s?", err, suggestion)
		}
	}
	if m := invalidArgument.FindStringSubmatch(err.Error()); m != nil {
		err = fmt.Errorf("invalid value %s for %s", m[1], m[2])
	}
	if m := shorthandIn.FindStringSubmatchIndex(err.Error()); m != nil {
		msg := err.Error()
		err = errors.New(msg[:m[0]] + "-" + msg[m[2]:m[3]])
	}
	if name, ok := strings.CutPrefix(err.Error(), "flag needs an argument: --"); ok && cmd.InheritedFlags().Lookup(name) != nil {
		return &FlagError{Err: err, Root: true}
	}
	return &FlagError{Err: err}
}

// invalidArgument matches pflag's error for a value its flag rejects, such as
// `invalid argument "x" for "-n, --names" flag: strconv.ParseBool: ...`.
var invalidArgument = regexp.MustCompile(`^invalid argument (".*") for "(?:-\w, )?(--[^"]+)" flag: `)

// shorthandIn matches the end of pflag's errors about one shorthand flag in
// a group, such as `unknown shorthand flag: 'x' in -xv`.
var shorthandIn = regexp.MustCompile(`'(.)' in -\S+$`)

// UnknownSubcommandError is the usage error for an unknown subcommand of
// cmd, with the closest subcommands as a suggestion.
func UnknownSubcommandError(cmd *cobra.Command, name string) error {
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	path := strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
	return FlagErrorf("unknown %s subcommand %q%s", path, name, DidYouMean(cmd.SuggestionsFor(name)))
}

// DidYouMean returns `; did you mean "a" or "b"?` for the suggested names,
// or "" when there are none.
func DidYouMean(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	quoted := make([]string, len(suggestions))
	for i, s := range suggestions {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "; did you mean " + strings.Join(quoted, " or ") + "?"
}

// closestFlag returns the name of cmd's flag nearest to name, or "" when none
// is within two edits.
func closestFlag(cmd *cobra.Command, name string) string {
	best, bestDistance := "", 3
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
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
