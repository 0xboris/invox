# invox customer list

```text
List all customers from customers.yaml.

Default lookup:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  schema/docs: run `invox help customers`

Usage:
  invox customer list [flags]

Flags:
  -c, --customers string   Path to customers.yaml
      --json fields        Output JSON with the specified fields

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

JSON fields:
  currency, email, id, name, status

Examples:
  $ invox customer list
  $ invox customer list -c customers.yaml
  $ invox customer list --json id,email
```

## See also

- [invox customer](invox_customer.md): Customer-related commands
