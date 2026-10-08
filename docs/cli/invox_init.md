# invox init

```text
Create starter support files in the global config directory.

Behavior:
  Creates the global config directory if it does not exist yet.
  Writes starter versions of config.yaml, customers.yaml, issuer.yaml,
  invoice_defaults.yaml, and template.tex.
  Existing non-empty files are left unchanged.
  When the deprecated invoice-tool directory has files the config directory
  lacks, asks first, then copies them in before writing the starter files.
  Nothing is replaced, and the invoice-tool directory is left in place.

Config directory:
  $HOME/.config/invox

Usage:
  invox init [flags]

Flags:
      --force   Copy files from the deprecated config directory without asking (required without a terminal)

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox init
  $ invox init --force
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
