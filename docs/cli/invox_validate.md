# invox validate

```text
Validate invoice YAML against customers and issuer data.

Required inputs:
  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file

Default lookup:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  schema/docs: run `invox help customers`
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml
  schema/docs: run `invox help issuer`

JSON output:
  --json prints one object, also when the invoice is invalid; invox then
  exits 1. The invoice's fields are null when it is invalid, and errors
  lists each problem with its file, line and field where they are known.

Usage:
  invox validate [INVOICE.yaml] [flags]

Flags:
  -c, --customers string   Path to customers.yaml
  -i, --input string       Input invoice YAML file
  -u, --issuer string      Path to issuer.yaml
      --json fields        Output JSON with the specified fields

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

JSON fields:
  currency, customerId, errors, lineItems, number, total, valid

Examples:
  $ invox validate invoice.yaml
  $ invox validate invoices/2026-0021.yaml -c customers.yaml -u issuer.yaml
  $ invox validate invoice.yaml --json valid,total,currency,errors
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
