# invox new

```text
Create a new invoice YAML file with a generated number and prefilled defaults.

Required inputs:
  CUSTOMER_ID                  Required positional argument

Default output:
  <invoice.number>.yaml in the current directory

Default lookup:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  schema/docs: run `invox help customers`
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml
  schema/docs: run `invox help issuer`
  invoice_defaults.yaml: upward project search, then $HOME/.config/invox/invoice_defaults.yaml
  schema/docs: run `invox help defaults`
  archive.dir: config.yaml, then $HOME/.local/share/invox/invoices

Usage:
  invox new CUSTOMER_ID [flags]

Flags:
  -c, --customers string   Path to customers.yaml
  -e, --edit               Open the created invoice in your editor
      --from-last          Use the latest archived invoice for this customer as the source document
  -u, --issuer string      Path to issuer.yaml
  -o, --output string      Output YAML path
  -s, --source string      Path to invoice_defaults.yaml

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox new CUST-001
  $ invox new CUST-001 -e
  $ invox new CUST-001 --from-last
  $ invox new CUST-001 -o invoices/2026-0022.yaml -s invoice_defaults.yaml -c customers.yaml -u issuer.yaml
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
