// Package config is the `invox config` command, which opens config.yaml in
// the editor, and its subcommands.
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/config/edit"
	"github.com/0xboris/invox/internal/cmd/config/paths"
)

// NewCmdConfig returns the config command and its subcommands. Without a
// subcommand it runs config edit. runF replaces the run of config edit in
// tests.
func NewCmdConfig(f *cmdutil.Factory, runF func(context.Context, *edit.EditOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open config.yaml in your editor",
		Long: `Open config.yaml in your editor.

Behavior:
  Opens the resolved config.yaml in your editor, or the file given with
  --config.
  If config.yaml does not exist yet, creates it from the template below.
  Existing config.yaml files are left unchanged.

Config paths:
  default: {{.GlobalConfigPath}}
  --config PATH and INVOX_CONFIG_DIR change them; see ` + "`invox help environment`" + `.
  ` + "`invox config paths`" + ` shows the config file and the support files in use.

Formatting:
  Top-level keys must start at column 1 with no leading spaces.

Supported settings:
  paths.customers    Override the default customers.yaml lookup path
  paths.issuer       Override the default issuer.yaml lookup path
  paths.defaults     Override the default invoice_defaults.yaml lookup path
  paths.template     Override the default template.tex lookup path
  numbering.pattern  Override the invoice-number pattern
  numbering.start    Global starting counter when no archived invoice matches
  archive.dir        Override the archive directory for archived invoice files
  email.subject      Override the draft email subject template for the email command
  email.body         Override the plain-text body template for the email command

Invoice numbers:
  ` + "`new`" + ` uses the next counter after the highest one found in archive.dir and in
  draft or built invoice YAML files in the current directory and the output directory.
  ` + "`archive add`" + ` refuses an invoice whose number is already archived under another file;
  ` + "`validate`" + ` warns about it. Run ` + "`invox increment FILE`" + ` to give it the next free number.

email template placeholders:
` + emailPlaceholderList() + `  Any other {name} in email.subject, email.body or --subject fails the email command.

Customer overrides:
  customers.<CUSTOMER_ID>.numbering.start  Override numbering.start for one customer

Support file precedence:
  1. explicit CLI flag
  2. upward project search
  3. paths.* in config.yaml
  4. conventional files in {{.ConfigDir}}

Template:
{{.ConfigTemplate}}`,
		Example: `$ invox config
$ invox config edit
$ invox config paths
$ invox help config
`,
	}
	edit.Configure(cmd, f, runF)
	cmd.AddCommand(edit.NewCmdEdit(f, runF), paths.NewCmdPaths(f, nil))
	return cmd
}

// emailPlaceholderList lists the email placeholders, one per line.
func emailPlaceholderList() string {
	var b strings.Builder
	for _, p := range billing.EmailPlaceholders() {
		fmt.Fprintf(&b, "  %-22s %s\n", p.Name, p.Description)
	}
	return b.String()
}
