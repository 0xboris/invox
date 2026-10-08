# invox

```text
invox generates LaTeX and PDF invoices from YAML data.

Defaults:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml
  invoice_defaults.yaml: upward project search, then $HOME/.config/invox/invoice_defaults.yaml
  template.tex: upward project search, then $HOME/.config/invox/template.tex
  new output: ./<invoice.number>.yaml
  render output: ./invoice.tex
  email draft path: <input name>.eml in a new temporary directory, removed after 24 hours
  build output: input path with .pdf extension

Usage:
  invox <command> [flags]

Invoice commands:
  archive    Archive invoices, and list or edit archived ones
  build      Render and compile an invoice PDF with Tectonic
  email      Create an email draft and open it in the default mail app
  increment  Increment the invoice number in an existing invoice YAML file
  new        Create a new invoice YAML file with a generated number and prefilled defaults
  render     Render a LaTeX invoice file from YAML data
  validate   Validate invoice YAML against customers and issuer data

Setup commands:
  config    Open config.yaml in your editor
  customer  Customer-related commands
  init      Create starter support files in the global config directory
  template  Template-related commands

Additional commands:
  completion  Generate shell completion scripts
  version     Show the invox version

Help topics:
  config       config.yaml keys, precedence, and email placeholders
  customers    customers.yaml fields, aliases, and example
  issuer       issuer.yaml fields, validation rules, and example
  defaults     invoice_defaults.yaml shape and new-command behavior
  template     template placeholders and authoring rules
  environment  environment variables, default directories, and precedence
  exit-codes   what each exit status means

Flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead
      --version         Show the invox version

Examples:
  $ invox init
  $ invox customer list
  $ invox new CUST-001 -e
  $ invox new CUST-001 --from-last
  $ invox validate -i 2026-0001.yaml
  $ invox build 2026-0001.yaml --archive
  $ invox email 2026-0001.pdf
  $ invox archive edit 2026-0001.yaml

Learn more:
  Run `invox help <command>` for more information about a command.
  Run `invox help <topic>` to read a help topic.
```

## See also

- [invox archive](invox_archive.md): Archive invoices, and list or edit archived ones
- [invox build](invox_build.md): Render and compile an invoice PDF with Tectonic
- [invox completion](invox_completion.md): Generate shell completion scripts
- [invox config](invox_config.md): Open config.yaml in your editor
- [invox customer](invox_customer.md): Customer-related commands
- [invox email](invox_email.md): Create an email draft and open it in the default mail app
- [invox increment](invox_increment.md): Increment the invoice number in an existing invoice YAML file
- [invox init](invox_init.md): Create starter support files in the global config directory
- [invox new](invox_new.md): Create a new invoice YAML file with a generated number and prefilled defaults
- [invox render](invox_render.md): Render a LaTeX invoice file from YAML data
- [invox template](invox_template.md): Template-related commands
- [invox validate](invox_validate.md): Validate invoice YAML against customers and issuer data
- [invox version](invox_version.md): Show the invox version
- [invox help customers](invox_help_customers.md): customers.yaml fields, aliases, and example
- [invox help issuer](invox_help_issuer.md): issuer.yaml fields, validation rules, and example
- [invox help defaults](invox_help_defaults.md): invoice_defaults.yaml shape and new-command behavior
- [invox help environment](invox_help_environment.md): environment variables, default directories, and precedence
- [invox help exit-codes](invox_help_exit-codes.md): what each exit status means
