// Package config is the `invox config` command, which opens config.yaml in
// the editor, and its subcommands.
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cmd/config/paths"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

type ConfigOptions struct {
	IO     *iostreams.IOStreams
	Editor *editor.Editor
	Host   func() invoice.Host
	Getwd  func() (string, error)
}

// NewCmdConfig returns the config command and its subcommands. runF replaces
// configRun in tests.
func NewCmdConfig(f *cmdutil.Factory, runF func(context.Context, *ConfigOptions) error) *cobra.Command {
	opts := &ConfigOptions{IO: f.IOStreams, Editor: f.Editor, Host: f.Host, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:               "config",
		Short:             "Open config.yaml in your editor",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `Open config.yaml in your editor.

Behavior:
  Opens the resolved config.yaml in your editor, or the file given with
  --config.
  If config.yaml does not exist yet, creates it from the template below.
  Existing config.yaml files are left unchanged.

Config paths:
  preferred: {{.GlobalConfigPath}}
  legacy fallback: {{.LegacyConfigFile}}
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
  ` + "`archive`" + ` refuses an invoice whose number is already archived under another file;
  ` + "`validate`" + ` warns about it. Run ` + "`invox increment -i FILE`" + ` to give it the next free number.

email template placeholders:
  {customer_name}        Customer display name
  {email_greeting}       Customer-specific greeting, defaults to Hello,
  {contact_person}       Customer contact person
  {customer_id}          Customer ID from the invoice
  {invoice_number}       Invoice number
  {issue_date}           Invoice issue date
  {due_date}             Invoice due date
  {total_amount}         Invoice total with currency
  {outstanding_amount}   Outstanding amount with currency
  {payment_terms_text}   issuer.payment.payment_terms_text
  {issuer_name}          issuer.company.legal_company_name

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
$ invox config paths
$ invox help config
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("config", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(cmd.Context(), opts)
			}
			return configRun(cmd.Context(), opts)
		},
	}
	cmd.AddCommand(paths.NewCmdPaths(f, nil))
	return cmd
}

func configRun(ctx context.Context, opts *ConfigOptions) error {
	configPath, err := opts.Host().EditableConfigPath()
	if err != nil {
		return err
	}
	baseDir, err := opts.Getwd()
	if err != nil {
		return err
	}
	displayPath := invoice.DisplayPath(configPath, baseDir)

	if err := cmdutil.OpenInEditor(ctx, opts.IO, opts.Editor, "config", configPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "Opened %s\n", displayPath)
	return nil
}
