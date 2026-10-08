# invox email

```text
Create an email draft and open it in the default mail app.

Required inputs:
  INVOICE.yaml, INVOICE.pdf, or -i, --input PATH  Path to the invoice YAML or built PDF file

Default output:
  <input name>.eml in a new temporary directory, removed after 24 hours

Default lookup:
  customers.yaml: upward project search, then $HOME/.config/invox/customers.yaml
  schema/docs: run `invox help customers`
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml
  schema/docs: run `invox help issuer`
  invoice PDF: input path with .pdf extension by default, or the input itself when the input is a PDF

Behavior:
  Accepts either the invoice YAML file or the built PDF as input.
  When the input is a PDF, the matching YAML file is resolved from the same basename.
  The PDF lookup checks next to the PDF first, then archive.dir.
  Requires invoice.status to be built or archived and the PDF attachment to exist.
  On macOS, opens an editable compose window in Apple Mail with the PDF attached.
  If -o is set, or on non-macOS platforms, writes a .eml draft file and opens it.
  Without -o, the draft is written to a new temporary directory. A later run removes it after 24 hours.
  With -o, the draft is kept. An existing -o file is not replaced unless --force is set.
  Does not send the email and does not change invoice.status.

Usage:
  invox email [INVOICE.yaml | INVOICE.pdf] [flags]

Aliases:
  invox send

Flags:
  -c, --customers string   Path to customers.yaml
  -n, --dry-run            Print the recipient, subject and attachment, and neither write nor open a draft
      --force              Overwrite an existing output file
  -i, --input string       Input invoice YAML or PDF file
  -u, --issuer string      Path to issuer.yaml
  -o, --output string      Write the draft to this .eml file instead of a temporary one (must end with .eml)
  -p, --pdf string         Path to the invoice PDF (default: the input with .pdf)
      --subject string     Email subject override, supports placeholders
      --to string          Recipient email override

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox email invoice.yaml
  $ invox email invoice.pdf
  $ invox email invoice.yaml --to billing@example.com
  $ invox email invoice.yaml --dry-run
  $ invox email invoices/2026-0021.yaml -p out/2026-0021.pdf -o drafts/2026-0021.eml -c customers.yaml -u issuer.yaml
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
