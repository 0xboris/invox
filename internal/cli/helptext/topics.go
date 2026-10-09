package helptext

import (
	"fmt"
	"io"
)

// Topic is a page of `invox help NAME`. Print is nil for a topic that is the
// help of the command with the same name.
type Topic struct {
	Name    string
	Aliases []string
	Short   string
	Print   func(w io.Writer, l Locations)
}

// Topics lists the help topics in the order the root help shows them.
var Topics = []Topic{
	{Name: "config", Short: "config.yaml keys, precedence, and email placeholders"},
	{Name: "customers", Short: "customers.yaml fields, aliases, and example", Print: printCustomersHelp},
	{Name: "issuer", Short: "issuer.yaml fields, validation rules, and example", Print: printIssuerHelp},
	{Name: "defaults", Aliases: []string{"invoice-defaults", "invoice_defaults"}, Short: "invoice_defaults.yaml shape and new-command behavior", Print: printDefaultsHelp},
	{Name: "template", Short: "template placeholders and authoring rules"},
	{Name: "environment", Short: "environment variables, default directories, and precedence", Print: printEnvironmentHelp},
	{Name: "exit-codes", Short: "what each exit status means", Print: func(w io.Writer, _ Locations) { printExitCodesHelp(w) }},
}

// LookupTopic returns the topic called name or one of its aliases.
func LookupTopic(name string) (Topic, bool) {
	for _, topic := range Topics {
		if topic.Name == name {
			return topic, true
		}
		for _, alias := range topic.Aliases {
			if alias == name {
				return topic, true
			}
		}
	}
	return Topic{}, false
}

func commandExample(args string) string {
	return commandName + " " + args
}

func printCustomersHelp(w io.Writer, l Locations) {
	fmt.Fprintf(w, "customers.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help customers\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported customers.yaml shape used by new, validate, render, build, and email.\n")
	fmt.Fprintf(w, "  `invox customer edit` opens the resolved file for editing.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level customer IDs must start at column 1 with no leading spaces.\n\n")
	printCustomerFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  Preferred display name path is <customer>.name; <customer>.legal_company_name is also accepted.\n")
	fmt.Fprintf(w, "  Email lookup order is billing.send_invoice_to, billing.email, then email.\n")
	fmt.Fprintf(w, "  email_greeting defaults to Hello, when omitted.\n")
	fmt.Fprintf(w, "  billing.currency defaults to EUR.\n")
	fmt.Fprintf(w, "  numbering.code feeds {customer_code}; numbering.start overrides config.numbering.start for one customer.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n\n", l.Customers)
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help customers"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer edit"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -c customers.yaml"))
	printCustomerYAMLExample(w)
}

func printIssuerHelp(w io.Writer, l Locations) {
	fmt.Fprintf(w, "issuer.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help issuer\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported issuer.yaml shape used by new, validate, render, build, and email.\n")
	fmt.Fprintf(w, "  `invox init` writes a starter issuer.yaml with this structure.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level keys must start at column 1 with no leading spaces.\n\n")
	printIssuerFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  payment.due_days must be a non-negative integer.\n")
	fmt.Fprintf(w, "  payment.vat_label defaults to VAT when omitted.\n")
	fmt.Fprintf(w, "  payment.epc_qr.name defaults to company.legal_company_name.\n")
	fmt.Fprintf(w, "  payment.epc_qr.text defaults to invoice.number.\n")
	fmt.Fprintf(w, "  payment.epc_qr.label defaults to Pay via EPC-QR.\n")
	fmt.Fprintf(w, "  EPC QR generation requires a valid SEPA-scope payment.iban.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n\n", l.Issuer)
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help issuer"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -u issuer.yaml"))
	printIssuerYAMLExample(w)
}

func printDefaultsHelp(w io.Writer, l Locations) {
	fmt.Fprintf(w, "invoice_defaults.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help defaults\n", commandName)
	fmt.Fprintf(w, "  %s help invoice-defaults\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported invoice_defaults.yaml shape used by `invox new`.\n")
	fmt.Fprintf(w, "  `invox init` writes a starter invoice_defaults.yaml with this structure.\n")
	fmt.Fprintf(w, "  `invox new --from-last` bypasses invoice_defaults.yaml and clones the latest archived invoice for that customer.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level keys must start at column 1 with no leading spaces.\n\n")
	printInvoiceDefaultsFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  `new` sets customer_id, invoice.number, invoice.issue_date, invoice.due_date, invoice.status, and invoice.paid_amount.\n")
	fmt.Fprintf(w, "  If positions is omitted, `new` creates an empty list.\n")
	fmt.Fprintf(w, "  The final invoice used by validate/render/build/email still needs a non-empty positions list.\n")
	fmt.Fprintf(w, "  Canonical keys are positions, invoice.period, and invoice.vat_percent.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n\n", l.Defaults)
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help defaults"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 --defaults invoice_defaults.yaml"))
	printInvoiceDefaultsYAMLExample(w)
}

// environmentVariable is one entry of `invox help environment`. Every key the
// code reads with os.Getenv, os.LookupEnv or env.Env's Getenv must have one; a
// test checks it.
type environmentVariable struct {
	name        string
	description []string
}

var environmentVariables = []environmentVariable{
	{"INVOX_CONFIG_DIR", []string{
		"Config directory to use in place of the default one, on every OS. invox",
		"reads config.yaml and the global support files there, and `init` writes",
		"there. It must exist. --config still wins for config.yaml.",
	}},
	{"XDG_CONFIG_HOME", []string{
		"Base directory for the config directory, on every OS.",
		"Default: $HOME/.config. A relative value is ignored.",
	}},
	{"XDG_DATA_HOME", []string{
		"Base directory for the default archive directory on Linux, macOS and other",
		"Unix systems. Default: $HOME/.local/share, or $HOME/Library/Application",
		"Support on macOS. Not used on Windows. A relative value is ignored.",
	}},
	{"APPDATA", []string{
		"Base directory for the default archive directory on Windows.",
		"Default: %USERPROFILE%\\AppData\\Roaming. Not used elsewhere.",
		"A relative value is ignored.",
	}},
	{"VISUAL", []string{
		"Editor for `config`, `customer edit` and `new -e`. Wins over EDITOR.",
		"The value is split into words like a shell would and run directly, with the",
		"file as the last argument. A value with shell syntax ($, |, ; and the like)",
		"runs through sh -c, where an unquoted # starts a comment that drops the file.",
		"On Windows it is always split and run directly.",
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

func printEnvironmentHelp(w io.Writer, l Locations) {
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
	fmt.Fprintf(w, "  Linux:     $XDG_CONFIG_HOME/invox, else $HOME/.config/invox\n")
	fmt.Fprintf(w, "  macOS:     $XDG_CONFIG_HOME/invox, else $HOME/.config/invox\n")
	fmt.Fprintf(w, "  Windows:   %%XDG_CONFIG_HOME%%\\invox, else %%USERPROFILE%%\\.config\\invox\n")
	fmt.Fprintf(w, "  INVOX_CONFIG_DIR replaces it on every OS.\n")
	fmt.Fprintf(w, "  here:      %s\n\n", l.ConfigDir)
	fmt.Fprintf(w, "Default archive directory (when config.yaml sets no archive.dir):\n")
	fmt.Fprintf(w, "  Linux:     $XDG_DATA_HOME/invox/invoices, else $HOME/.local/share/invox/invoices\n")
	fmt.Fprintf(w, "  macOS:     $XDG_DATA_HOME/invox/invoices, else $HOME/Library/Application Support/invox/invoices\n")
	fmt.Fprintf(w, "  Windows:   %%APPDATA%%\\invox\\invoices, else %%USERPROFILE%%\\AppData\\Roaming\\invox\\invoices\n")
	fmt.Fprintf(w, "  here:      %s\n\n", l.ArchiveDir)
	fmt.Fprintf(w, "Config file:\n")
	fmt.Fprintf(w, "  1. --config PATH\n")
	fmt.Fprintf(w, "  2. config.yaml in INVOX_CONFIG_DIR\n")
	fmt.Fprintf(w, "  3. config.yaml in the config directory\n")
	fmt.Fprintf(w, "  A --config file that is missing or broken is an error. invox never falls\n")
	fmt.Fprintf(w, "  back to another config file.\n\n")
	fmt.Fprintf(w, "Precedence:\n")
	fmt.Fprintf(w, "  An explicit flag wins, then config.yaml, then the defaults above. Environment\n")
	fmt.Fprintf(w, "  variables only move the default directories; no variable overrides a flag or\n")
	fmt.Fprintf(w, "  a config.yaml setting.\n\n")
	fmt.Fprintf(w, "Support file resolution (customers.yaml, issuer.yaml, invoice_defaults.yaml, template):\n")
	fmt.Fprintf(w, "  1. explicit flag (-c, -u, --defaults, -t)\n")
	fmt.Fprintf(w, "  2. upward search from the current directory\n")
	fmt.Fprintf(w, "  3. paths.* in config.yaml, relative to config.yaml\n")
	fmt.Fprintf(w, "  4. the file in the config directory\n\n")
	fmt.Fprintf(w, "Upward search:\n")
	fmt.Fprintf(w, "  It always searches the current directory. It goes up to the nearest directory\n")
	fmt.Fprintf(w, "  that holds .git, invox.yaml or invoice_defaults.yaml, and stops below your\n")
	fmt.Fprintf(w, "  home directory, which it searches only when invox runs there. Outside your\n")
	fmt.Fprintf(w, "  home directory, without such a marker above, it searches only the current\n")
	fmt.Fprintf(w, "  directory.\n\n")
	fmt.Fprintf(w, "Archive directory resolution:\n")
	fmt.Fprintf(w, "  1. archive.dir in config.yaml, relative to config.yaml\n")
	fmt.Fprintf(w, "  2. the default archive directory above\n\n")
	fmt.Fprintf(w, "Run `%s config paths` to see what each lookup finds.\n\n", commandName)
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
