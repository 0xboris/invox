package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	configcmd "github.com/0xboris/invox/internal/cmd/config"
	customercmd "github.com/0xboris/invox/internal/cmd/customer"
	initcmd "github.com/0xboris/invox/internal/cmd/init"
	templatecmd "github.com/0xboris/invox/internal/cmd/template"
	versioncmd "github.com/0xboris/invox/internal/cmd/version"
)

// newRootCmd returns the invox command tree. Commands that have not moved to
// cobra yet reach the root's RunE, which hands its unparsed arguments to the
// legacy dispatcher. Main prints errors and picks exit codes, so cobra prints
// neither errors nor usage.
func newRootCmd(f *cmdutil.Factory) *cobra.Command {
	ios := f.IOStreams
	root := &cobra.Command{
		Use:                "invox",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		CompletionOptions:  cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyGlobalFlags(cmd, f)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLegacy(cmd.Context(), f, args)
		},
	}
	root.SuggestionsMinimumDistance = 2
	root.SetIn(ios.In)
	root.SetOut(ios.Out)
	root.SetErr(ios.ErrOut)
	root.SetFlagErrorFunc(cmdutil.FlagErrorFunc)

	root.PersistentFlags().String("config", "", "Read this config file instead of config.yaml")
	root.PersistentFlags().Bool("no-input", false, "Never prompt or open an editor")

	// Until help is generated from the command tree (#48), every help request
	// prints the hand-written page for the command.
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if err := runHelp(f, strings.Fields(cmdutil.CommandPath(cmd))); err != nil {
			fmt.Fprintf(ios.ErrOut, "error: %s\n", err)
		}
	})
	root.SetHelpCommand(&cobra.Command{
		Use:                "help",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLegacy(cmd.Context(), f, append([]string{"help"}, args...))
		},
	})

	root.AddCommand(
		configcmd.NewCmdConfig(f, nil),
		customercmd.NewCmdCustomer(f),
		initcmd.NewCmdInit(f, nil),
		templatecmd.NewCmdTemplate(f),
		versioncmd.NewCmdVersion(f, nil),
	)
	return root
}

// applyGlobalFlags copies the global flags cmd parsed into f.
func applyGlobalFlags(cmd *cobra.Command, f *cmdutil.Factory) error {
	flags := cmd.Flags()
	if flags.Changed("config") {
		configFile, _ := flags.GetString("config")
		if configFile == "" {
			return cmdutil.FlagErrorf("", "flag needs an argument: --config")
		}
		f.ConfigFile = configFile
	}
	if noInput, _ := flags.GetBool("no-input"); noInput {
		f.IOStreams.SetNeverPrompt(true)
	}
	return nil
}

// versionFlagToCommand turns `invox --version` into `invox version`, keeping
// global flags in front of it.
func versionFlagToCommand(args []string) []string {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--no-input", strings.HasPrefix(args[i], "--config="):
		case args[i] == "--config":
			i++
		case args[i] == "--version":
			out := append([]string{}, args...)
			out[i] = "version"
			return out
		default:
			return args
		}
	}
	return args
}

// normalizeLongFlags rewrites the single-dash long flags of a cobra command,
// such as -names, to their double-dash form, and warns about each on w. The
// legacy dispatcher's flag package accepts both forms, so its arguments are
// left alone.
func normalizeLongFlags(root *cobra.Command, args []string, w io.Writer) []string {
	cmd, _, err := root.Find(args)
	if err != nil || cmd.DisableFlagParsing {
		return args
	}
	cmd.InitDefaultHelpFlag()
	lookup := func(name string) *pflag.Flag {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			return flag
		}
		return cmd.InheritedFlags().Lookup(name)
	}
	takesValue := func(flag *pflag.Flag, arg string) bool {
		return flag != nil && flag.NoOptDefVal == "" && !strings.Contains(arg, "=")
	}

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(out, args[i:]...)
		}
		var flag *pflag.Flag
		switch {
		case strings.HasPrefix(arg, "--"):
			name, _, _ := strings.Cut(arg[2:], "=")
			flag = lookup(name)
		case len(arg) == 2 && arg[0] == '-':
			flag = cmd.Flags().ShorthandLookup(arg[1:])
			if flag == nil {
				flag = cmd.InheritedFlags().ShorthandLookup(arg[1:])
			}
		case len(arg) > 2 && arg[0] == '-':
			name, _, _ := strings.Cut(arg[1:], "=")
			if flag = lookup(name); flag != nil {
				fmt.Fprintf(w, "warning: -%s is deprecated; use --%s\n", name, name)
				arg = "-" + arg
			} else if isLetter(arg[1]) && cmd.Flags().ShorthandLookup(arg[1:2]) == nil && cmd.InheritedFlags().ShorthandLookup(arg[1:2]) == nil {
				// Not a shorthand cluster such as -ofile.yaml: report it as
				// the unknown long flag it looks like, with a suggestion.
				arg = "-" + arg
			}
		}
		out = append(out, arg)
		if takesValue(flag, arg) && i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

func isLetter(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}
