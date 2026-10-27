package cmdutil

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"
)

// DeprecatedFlagAnnotation marks a flag that warns about itself when used,
// so the single-dash normaliser adds no warning of its own.
const DeprecatedFlagAnnotation = "deprecated"

// DeprecateFlag hides the flag name in flags and marks it deprecated. The
// command that owns it prints the warning.
func DeprecateFlag(flags *pflag.FlagSet, name string) {
	_ = flags.MarkHidden(name)
	_ = flags.SetAnnotation(name, DeprecatedFlagAnnotation, []string{"true"})
}

// WarnDeprecated prints the one-line warning that old is deprecated and new
// replaces it.
func WarnDeprecated(w io.Writer, old, new string) {
	fmt.Fprintf(w, "warning: %s is deprecated; use %s\n", old, new)
}
