package cmdutil

import (
	"fmt"
	"io"
)

// WarnDeprecated prints the one-line warning that old is deprecated and new
// replaces it.
func WarnDeprecated(w io.Writer, old, new string) {
	fmt.Fprintf(w, "warning: %s is deprecated; use %s\n", old, new)
}
