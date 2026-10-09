package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	completioncmd "github.com/0xboris/invox/internal/cmd/completion"
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
	root := &cobra.Command{
		Use:   "invox",
		Short: "Generate LaTeX and PDF invoices from YAML data",
		Long: `invox generates LaTeX and PDF invoices from YAML data.

Defaults:
  customers.yaml: upward project search, then {{.Customers}}
  issuer.yaml: upward project search, then {{.Issuer}}
  invoice_defaults.yaml: upward project search, then {{.Defaults}}
  template.tex: upward project search, then {{.Template}}
  new output: ./<invoice.number>.yaml
  render output: ./invoice.tex
  email draft path: <input name>.eml in a new temporary directory, removed after 24 hours
  build output: input path with .pdf extension
`,
		Example: `$ invox init
$ invox customer list
$ invox new CUST-001 -e
$ invox new CUST-001 --from-last
$ invox validate -i 2026-0001.yaml
$ invox build 2026-0001.yaml --archive
$ invox email 2026-0001.pdf
$ invox archive edit 2026-0001.yaml
`,
		Args:              cobra.ArbitraryArgs,
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyGlobalFlags(cmd, f)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmdutil.FlagErrorf("missing subcommand")
			}
			return unknownSubcommand(cmd, args[0])
		},
	}
	root.Flags().Bool("version", false, "Show the invox version")
	root.SuggestionsMinimumDistance = 2
	root.SetIn(ios.In)
	root.SetOut(ios.Out)
	root.SetErr(ios.ErrOut)
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		// In `invox send --to x`, the unknown subcommand is the error.
		if cmd == root {
			if rest := cmd.Flags().Args(); len(rest) > 0 {
				return unknownSubcommand(cmd, rest[0])
			}
		}
		return cmdutil.FlagErrorFunc(cmd, err)
	})

	root.PersistentFlags().String("config", "", "Read this config file instead of config.yaml")
	root.PersistentFlags().Bool("no-input", false, "Never prompt or open an editor; fail with exit 2 instead")

	root.PersistentFlags().BoolP("help", "h", false, "Show help for a command")

	help := &cobra.Command{
		Use:    "help [command | topic]",
		Hidden: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return helpCompletions(root, args), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return helpTopic(cmd.OutOrStdout(), root, f.Locations(), args)
		},
	}
	root.SetHelpCommand(help)
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// `invox nope --help` is still an unknown subcommand.
		if cmd == root {
			if rest := cmd.Flags().Args(); len(rest) > 0 {
				helpErr = unknownSubcommand(cmd, rest[0])
				return
			}
		}
		// `invox help --help` is the root help.
		if cmd == help {
			cmd = root
		}
		if err := writeHelp(cmd.OutOrStdout(), cmd, f.Locations()); err != nil {
			helpErr = err
		}
	})

	root.AddGroup(
		&cobra.Group{ID: "invoice", Title: "Invoice commands"},
		&cobra.Group{ID: "setup", Title: "Setup commands"},
	)
	root.AddCommand(
		archivecmd.NewCmdArchive(f),
		configcmd.NewCmdConfig(f, nil),
		buildcmd.NewCmdBuild(f, nil),
		completioncmd.NewCmdCompletion(f, nil),
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
	for _, cmd := range root.Commands() {
		cmd.GroupID = commandGroups[cmd.Name()]
	}
	return root, func() error { return helpErr }
}

// commandGroups puts the root's commands into the groups its help lists.
// A command with no entry is listed under "Additional commands".
var commandGroups = map[string]string{
	"new":       "invoice",
	"increment": "invoice",
	"validate":  "invoice",
	"render":    "invoice",
	"build":     "invoice",
	"email":     "invoice",
	"archive":   "invoice",
	"init":      "setup",
	"config":    "setup",
	"customer":  "setup",
	"template":  "setup",
}

// unknownSubcommand is the usage error for name on the root, with the closest
// subcommands as a suggestion. A real subcommand only reaches the root after
// `--`, which the user typed where the subcommand belongs.
func unknownSubcommand(root *cobra.Command, name string) error {
	if cmd, _, err := root.Find([]string{name}); err == nil && cmd != root {
		return cmdutil.FlagErrorf("unknown subcommand %q", "--")
	}
	message := fmt.Sprintf("unknown subcommand %q", name) + cmdutil.DidYouMean(root.SuggestionsFor(name))
	return cmdutil.FlagErrorf("%s", message)
}

// applyGlobalFlags copies the global flags cmd parsed into f.
func applyGlobalFlags(cmd *cobra.Command, f *cmdutil.Factory) error {
	flags := cmd.Flags()
	if flags.Changed("config") {
		configFile, _ := flags.GetString("config")
		if configFile == "" {
			return &cmdutil.FlagError{Err: errors.New("flag needs an argument: --config"), Root: true}
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

// checkSingleDashFlags returns the usage error for the first single-dash word
// of three or more characters in args that pflag would misread as a group of
// shorthand flags: one naming a long flag of the command args run, such as
// -names, or one whose first letter is no shorthand. pflag would read
// -output=x.pdf as -o utput=x.pdf. A real group such as -ofile.yaml passes.
// It also returns the command args run, whose help explains the error.
func checkSingleDashFlags(root *cobra.Command, args []string) (*cobra.Command, error) {
	// Find fails only on an unknown command, which cobra reports when it runs
	// args, and it still returns the command it got to.
	cmd, _, _ := root.Find(args)
	cmd.InitDefaultHelpFlag()
	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.InheritedFlags())

	for i := 0; i < len(args); i++ {
		arg := args[i]
		var flag *pflag.Flag
		switch {
		case arg == "--":
			return cmd, nil
		case cmd == root && !strings.HasPrefix(arg, "-"):
			// An unknown subcommand, which cobra reports.
			return cmd, nil
		case strings.HasPrefix(arg, "--"):
			name, _, _ := strings.Cut(arg[2:], "=")
			flag = flags.Lookup(name)
		case len(arg) == 2 && arg[0] == '-':
			flag = flags.ShorthandLookup(arg[1:])
		case len(arg) > 2 && arg[0] == '-' && isLetter(arg[1]):
			name, _, _ := strings.Cut(arg[1:], "=")
			if flags.Lookup(name) != nil {
				return cmd, cmdutil.FlagErrorf("-%s is not a flag; use --%s", name, name)
			}
			if flags.ShorthandLookup(arg[1:2]) == nil {
				return cmd, cmdutil.FlagErrorFunc(cmd, fmt.Errorf("unknown flag: -%s", name))
			}
		}
		if flag != nil && flag.NoOptDefVal == "" && !strings.Contains(arg, "=") {
			i++
		}
	}
	return cmd, nil
}

func isLetter(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}

// NewRootCmd returns the invox command tree for f, the one Main runs. The
// docs generator walks it.
func NewRootCmd(f *cmdutil.Factory) *cobra.Command {
	root, _ := newRootCmd(f)
	return root
}
