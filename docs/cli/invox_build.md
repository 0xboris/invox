# invox build

```text
Render and compile an invoice PDF with Tectonic.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default output:
  the input path with .pdf extension

Default lookup:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  schema/docs: run `invox help customers`
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml
  schema/docs: run `invox help issuer`
  template.tex: upward project search, then $HOME/.config/invox/template.tex

Replacing an archived invoice:
  An invoice with invoice.status archived keeps that status when its PDF is rebuilt.
  With --archive, a working copy from `invox archive edit` replaces the archived invoice it came from.
  On a terminal you are asked to confirm; otherwise pass --yes. Declining exits with status 2.
  The previous version is kept as archive.dir/.history/<path>.<UTC timestamp>.<ext>,
  which archive list, numbering and the duplicate-number check ignore.
  --yes only answers the question; every other check still applies.

Usage:
  invox build [INVOICE.yaml] [flags]

Flags:
      --archive            Archive the invoice after a successful build
  -c, --customers string   Path to customers.yaml
  -i, --input string       Input invoice YAML file
  -u, --issuer string      Path to issuer.yaml
  -o, --output string      Output PDF path (must end with .pdf; default: the input with .pdf)
  -t, --template string    Template path or name
      --yes                Replace an archived invoice without asking

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox build invoice.yaml
  $ invox build invoice.yaml --archive
  $ invox build invoices/2026-0021.yaml -o out/2026-0021.pdf -c customers.yaml -u issuer.yaml -t template.tex
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
