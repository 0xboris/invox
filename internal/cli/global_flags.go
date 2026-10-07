package cli

import (
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
)

// splitGlobalFlags removes --config PATH and --config=PATH from args, anywhere
// before a bare "--", and returns the last value given. A --config without a
// value is a usage error.
func splitGlobalFlags(args []string) (configFile string, rest []string, err error) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return configFile, append(rest, args[i:]...), nil
		case arg == "--config":
			if i+1 >= len(args) || args[i+1] == "" {
				return "", nil, cmdutil.FlagErrorf("", "flag needs an argument: --config")
			}
			i++
			configFile = args[i]
		case strings.HasPrefix(arg, "--config="):
			configFile = strings.TrimPrefix(arg, "--config=")
			if configFile == "" {
				return "", nil, cmdutil.FlagErrorf("", "flag needs an argument: --config")
			}
		default:
			rest = append(rest, arg)
		}
	}
	return configFile, rest, nil
}
