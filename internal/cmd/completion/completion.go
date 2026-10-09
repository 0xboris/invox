// Package completion is the `invox completion` command.
package completion

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// shells maps each shell `invox completion` supports to the cobra generator
// of its script.
var shells = map[string]func(root *cobra.Command, w io.Writer) error{
	"bash":       func(root *cobra.Command, w io.Writer) error { return root.GenBashCompletionV2(w, true) },
	"zsh":        (*cobra.Command).GenZshCompletion,
	"fish":       func(root *cobra.Command, w io.Writer) error { return root.GenFishCompletion(w, true) },
	"powershell": (*cobra.Command).GenPowerShellCompletionWithDesc,
}

type CompletionOptions struct {
	IO *iostreams.IOStreams
	// Root is the command tree the script completes.
	Root  *cobra.Command
	Shell string
}

// NewCmdCompletion returns the completion command. Without a shell it
// prints its help. runF replaces completionRun in tests.
func NewCmdCompletion(f *cmdutil.Factory, runF func(*CompletionOptions) error) *cobra.Command {
	opts := &CompletionOptions{IO: f.IOStreams}
	return &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts.

Supported shells:
  bash, zsh, fish, powershell

Notes:
  The script completes commands and flags, customer IDs for new, template
  names for -t/--template, archived invoices for archive edit, and file names.
  It asks invox for the values each time, so they follow your files. When a
  file is missing or broken, the values from it are not completed.

Bash:
  source <(invox completion bash)
  Persistent install, with the bash-completion package:
    invox completion bash > ~/.local/share/bash-completion/completions/invox

Zsh:
  source <(invox completion zsh)
  Persistent install:
    mkdir -p ~/.zsh/completions
    invox completion zsh > ~/.zsh/completions/_invox
  Add this before compinit in ~/.zshrc:
    fpath=(~/.zsh/completions $fpath)
    autoload -Uz compinit
    compinit

Fish:
  invox completion fish > ~/.config/fish/completions/invox.fish

PowerShell:
  invox completion powershell | Out-String | Invoke-Expression
  Add that line to your $PROFILE to load it in every session.
`,
		Example: `$ invox completion zsh
$ invox completion bash > ~/.local/share/bash-completion/completions/invox
`,
		ValidArgs: []cobra.Completion{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				return cmd.Help()
			case len(args) > 1:
				return cmdutil.FlagErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if _, ok := shells[args[0]]; !ok {
				return cmdutil.FlagErrorf("unsupported shell %q", args[0])
			}
			opts.Root = cmd.Root()
			opts.Shell = args[0]
			if runF != nil {
				return runF(opts)
			}
			return completionRun(opts)
		},
	}
}

func completionRun(opts *CompletionOptions) error {
	return shells[opts.Shell](opts.Root, opts.IO.Out)
}
