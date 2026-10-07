package cli

import (
	"fmt"
	"io"

	"github.com/0xboris/invox/internal/invoice"
)

// environmentVariable is one entry of `invox help environment`. Every key the
// code reads with os.Getenv, os.LookupEnv or env.Env's Getenv must have one; a
// test checks it.
type environmentVariable struct {
	name        string
	description []string
}

var environmentVariables = []environmentVariable{
	{"XDG_CONFIG_HOME", []string{
		"Base directory for the config directory, on every OS.",
		"Default: $HOME/.config.",
	}},
	{"XDG_DATA_HOME", []string{
		"Base directory for the default archive directory on Linux and other Unix systems.",
		"Default: $HOME/.local/share. Not used on macOS or Windows.",
	}},
	{"APPDATA", []string{
		"Base directory for the default archive directory on Windows.",
		"Default: %USERPROFILE%\\AppData\\Roaming. Not used elsewhere.",
	}},
	{"VISUAL", []string{
		"Editor for `config`, `customer config` and `new -e`. Wins over EDITOR.",
		"The value is split into words like a shell would and run directly, with the",
		"file as the last argument. A value with shell syntax ($, |, ; and the like)",
		"runs through sh -c. On Windows it is always split and run directly.",
	}},
	{"EDITOR", []string{
		"Editor used when VISUAL is unset, read the same way. Without either: vi, or",
		"notepad on Windows.",
	}},
	{"INVOX_PROMPT_DISABLED", []string{
		"Any non-empty value works like --no-input. With either, invox never prompts or",
		"opens an editor, and a step that needs one fails with exit 2.",
	}},
	{"INVOX_FORCE_TTY", []string{
		"Testing aid: any non-empty value makes invox treat stdout as a terminal.",
	}},
}

func printEnvironmentHelp(w io.Writer, h invoice.Host) {
	fmt.Fprintf(w, "Environment variables and default directories.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help environment\n\n", commandName)
	fmt.Fprintf(w, "Environment variables:\n")
	for _, variable := range environmentVariables {
		fmt.Fprintf(w, "  %s\n", variable.name)
		for _, line := range variable.description {
			fmt.Fprintf(w, "      %s\n", line)
		}
	}
	fmt.Fprintf(w, "\nConfig directory (config.yaml and the global support files):\n")
	fmt.Fprintf(w, "  all OSes:  $XDG_CONFIG_HOME/invox, else $HOME/.config/invox\n")
	fmt.Fprintf(w, "  legacy:    $XDG_CONFIG_HOME/invoice-tool, else $HOME/.config/invoice-tool,\n")
	fmt.Fprintf(w, "             read when a file is missing from the invox directory\n")
	fmt.Fprintf(w, "  here:      %s\n", h.ConfigDir())
	fmt.Fprintf(w, "  $HOME is %%USERPROFILE%% on Windows.\n\n")
	fmt.Fprintf(w, "Default archive directory (when config.yaml sets no archive.dir):\n")
	fmt.Fprintf(w, "  Linux:     $XDG_DATA_HOME/invox/invoices, else $HOME/.local/share/invox/invoices\n")
	fmt.Fprintf(w, "  macOS:     $HOME/Library/Application Support/invox/invoices\n")
	fmt.Fprintf(w, "  Windows:   %%APPDATA%%\\invox\\invoices, else %%USERPROFILE%%\\AppData\\Roaming\\invox\\invoices\n")
	fmt.Fprintf(w, "  here:      %s\n\n", h.DefaultArchiveDir())
	fmt.Fprintf(w, "Precedence:\n")
	fmt.Fprintf(w, "  An explicit flag wins, then config.yaml, then the defaults above. Environment\n")
	fmt.Fprintf(w, "  variables only move the default directories; no variable overrides a flag or\n")
	fmt.Fprintf(w, "  a config.yaml setting.\n\n")
	fmt.Fprintf(w, "Support file resolution (customers.yaml, issuer.yaml, invoice_defaults.yaml, template):\n")
	fmt.Fprintf(w, "  1. explicit flag (-c, -u, -s, -t)\n")
	fmt.Fprintf(w, "  2. upward search from the current directory to the filesystem root\n")
	fmt.Fprintf(w, "  3. paths.* in config.yaml, relative to config.yaml\n")
	fmt.Fprintf(w, "  4. the file in the config directory, then in the legacy directory\n\n")
	fmt.Fprintf(w, "Archive directory resolution:\n")
	fmt.Fprintf(w, "  1. archive.dir in config.yaml, relative to config.yaml\n")
	fmt.Fprintf(w, "  2. the default archive directory above\n\n")
	fmt.Fprintf(w, "See also:\n")
	fmt.Fprintf(w, "  %s help config\n", commandName)
}

func printExitCodesHelp(w io.Writer) {
	fmt.Fprintf(w, "Exit codes.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help exit-codes\n\n", commandName)
	fmt.Fprintf(w, "Exit codes:\n")
	fmt.Fprintf(w, "  0    Success.\n")
	fmt.Fprintf(w, "  1    The command failed: invalid invoice data, a missing or broken file,\n")
	fmt.Fprintf(w, "       a failed build or a failed external program.\n")
	fmt.Fprintf(w, "  2    Usage error: unknown command, topic or flag, or a missing or extra\n")
	fmt.Fprintf(w, "       argument. Also used when a confirmation was declined, or was needed\n")
	fmt.Fprintf(w, "       without a terminal to ask on and without --yes, and when an editor was\n")
	fmt.Fprintf(w, "       needed without a terminal or with --no-input. The step that needed\n")
	fmt.Fprintf(w, "       confirmation or the editor was not done. Ctrl-C at a confirmation\n")
	fmt.Fprintf(w, "       prompt also exits 2.\n")
	fmt.Fprintf(w, "  130  Interrupted by Ctrl-C (SIGINT). Programs invox started are stopped\n")
	fmt.Fprintf(w, "       and its temporary files are removed. While an editor runs, Ctrl-C\n")
	fmt.Fprintf(w, "       goes to the editor alone and invox keeps waiting for it.\n")
	fmt.Fprintf(w, "  143  Stopped by SIGTERM, with the same cleanup as 130.\n")
}
