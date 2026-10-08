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
	archivecmd "github.com/0xboris/invox/internal/cmd/invoice/archive"
	buildcmd "github.com/0xboris/invox/internal/cmd/invoice/build"
	emailcmd "github.com/0xboris/invox/internal/cmd/invoice/email"
	incrementcmd "github.com/0xboris/invox/internal/cmd/invoice/increment"
	newcmd "github.com/0xboris/invox/internal/cmd/invoice/new"
	rendercmd "github.com/0xboris/invox/internal/cmd/invoice/render"
	validatecmd "github.com/0xboris/invox/internal/cmd/invoice/validate"
	templatecmd "github.com/0xboris/invox/internal/cmd/template"
	versioncmd "github.com/0xboris/invox/internal/cmd/version"
)

// newRootCmd returns the invox command tree, and a func that returns the
// usage error a help request on the root found, since cobra's help funcs
// can't return one. Main prints errors and picks exit codes, so cobra prints
// neither errors nor usage.
func newRootCmd(f *cmdutil.Factory) (*cobra.Command, func() error) {
	ios := f.IOStreams
	var helpErr error
	var root *cobra.Command
	root = &cobra.Command{
		Use:               "invox",
		Args:              cobra.ArbitraryArgs,
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyGlobalFlags(cmd, f)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmdutil.FlagErrorf("", "missing subcommand")
			}
			return unknownSubcommand(cmd, args[0])
		},
	}
	root.Flags().Bool("version", false, "Show the invox version")
	_ = root.Flags().MarkHidden("version")
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
		// `invox nope --help` is still an unknown subcommand.
		if cmd == root {
			if rest := cmd.Flags().Args(); len(rest) > 0 {
				helpErr = unknownSubcommand(cmd, rest[0])
				return
			}
		}
		var topic []string
		if cmd.Name() != "help" {
			topic = strings.Fields(cmdutil.CommandPath(cmd))
		}
		if err := runHelp(f, topic); err != nil {
			fmt.Fprintf(ios.ErrOut, "error: %s\n", err)
		}
	})
	root.SetHelpCommand(&cobra.Command{
		Use:    "help [topic]",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHelp(f, args)
		},
	})

	root.AddCommand(
		archivecmd.NewCmdArchive(f, nil),
		configcmd.NewCmdConfig(f, nil),
		buildcmd.NewCmdBuild(f, nil),
		newCmdCompletion(f),
		customercmd.NewCmdCustomer(f),
		emailcmd.NewCmdEmail(f, nil),
		incrementcmd.NewCmdIncrement(f, nil),
		initcmd.NewCmdInit(f, nil),
		newcmd.NewCmdNew(f, nil),
		rendercmd.NewCmdRender(f, nil),
		templatecmd.NewCmdTemplate(f),
		validatecmd.NewCmdValidate(f, nil),
		versioncmd.NewCmdVersion(f, nil),
	)
	return root, func() error { return helpErr }
}

// unknownSubcommand is the usage error for name on the root, with the closest
// subcommands as a suggestion. A real subcommand only reaches the root after
// `--`, which the user typed where the subcommand belongs.
func unknownSubcommand(root *cobra.Command, name string) error {
	if cmd, _, err := root.Find([]string{name}); err == nil && cmd != root {
		return cmdutil.FlagErrorf("", "unknown subcommand %q", "--")
	}
	message := fmt.Sprintf("unknown subcommand %q", name)
	if suggestions := root.SuggestionsFor(name); len(suggestions) > 0 {
		message += fmt.Sprintf("; did you mean %q?", strings.Join(suggestions, `" or "`))
	}
	return cmdutil.FlagErrorf("", "%s", message)
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

// normalizeLongFlags rewrites the single-dash long flags of the command args
// run, such as -names, to their double-dash form, and warns about each on w.
func normalizeLongFlags(root *cobra.Command, args []string, w io.Writer) []string {
	cmd, _, err := root.Find(args)
	if err != nil {
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
