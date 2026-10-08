package cli

import (
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
)

// globalFlags are the flags every command accepts.
type globalFlags struct {
	configFile string
	noInput    bool
}

// splitGlobalFlags removes --config PATH, --config=PATH and --no-input from
// args, anywhere before a bare "--", and returns them. The last --config wins.
// A --config without a value is a usage error.
func splitGlobalFlags(args []string) (globalFlags, []string, error) {
	var g globalFlags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return g, append(rest, args[i:]...), nil
		case arg == "--no-input":
			g.noInput = true
		case arg == "--config":
			if i+1 >= len(args) || args[i+1] == "" {
				return globalFlags{}, nil, cmdutil.FlagErrorf("", "flag needs an argument: --config")
			}
			i++
			g.configFile = args[i]
		case strings.HasPrefix(arg, "--config="):
			g.configFile = strings.TrimPrefix(arg, "--config=")
			if g.configFile == "" {
				return globalFlags{}, nil, cmdutil.FlagErrorf("", "flag needs an argument: --config")
			}
		default:
			rest = append(rest, arg)
		}
	}
	return g, rest, nil
}
