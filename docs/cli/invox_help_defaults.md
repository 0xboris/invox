# invox help defaults

```text
invoice_defaults.yaml reference.

Usage:
  invox help defaults
  invox help invoice-defaults

Behavior:
  Shows the supported invoice_defaults.yaml shape used by `invox new`.
  `invox init` writes a starter invoice_defaults.yaml with this structure.
  `invox new --from-last` bypasses invoice_defaults.yaml and clones the latest archived invoice for that customer.

Formatting:
  Top-level keys must start at column 1 with no leading spaces.

invoice_defaults.yaml fields:
  invoice_defaults.yaml is the source document for `invox new`.
  The created invoice later also gains a top-level customer_id.

Top-level keys:
  invoice                             Mapping of invoice defaults used as the source document for `new`
  positions                           Line-item list copied into the created invoice; if omitted `new` creates an empty list

Invoice fields:
  invoice.number                      Usually blank in defaults; `new` always replaces it with the next generated invoice number
  invoice.issue_date                  Usually blank in defaults; `new` always replaces it with the current date
  invoice.due_date                    Usually blank in defaults; `new` always replaces it using issuer.payment.due_days
  invoice.status                      Usually `draft`; `new` always resets it to `draft`
  invoice.period                      Invoice period label copied into the created invoice and required by validate/render/build
  invoice.vat_percent                 Optional default VAT rate for the whole invoice; can be filled from customer.tax.default_vat_rate
  invoice.paid_amount                 Usually `0`; must be >= 0 and <= the invoice total; `new` always resets it to `0`

Position fields:
  positions[].name                    Line-item name
  positions[].description             Line-item description
  positions[].unit_price              Line-item net unit price; must be >= 0 on the final invoice
  positions[].quantity                Line-item quantity; must be > 0 on the final invoice
  positions[].vat_percent             Optional per-line VAT override

Unsupported legacy keys:
  line_items                          Unsupported; use positions
  invoice.period_label                Unsupported; use invoice.period
  invoice.vat_rate_percent            Unsupported; use invoice.vat_percent


Rules:
  `new` sets customer_id, invoice.number, invoice.issue_date, invoice.due_date, invoice.status, and invoice.paid_amount.
  If positions is omitted, `new` creates an empty list.
  The final invoice used by validate/render/build/email still needs a non-empty positions list.
  Canonical keys are positions, invoice.period, and invoice.vat_percent.

Lookup:
  invoice_defaults.yaml: upward project search, then $HOME/.config/invox/invoice_defaults.yaml

Examples:
  invox help defaults
  invox new CUST-001 --defaults invoice_defaults.yaml

invoice_defaults.yaml example:
invoice:
  number: ""
  issue_date: ""
  due_date: ""
  status: draft
  period: "Leistungszeitraum: "
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Example position
    description: Description of the delivered service
    unit_price: 100
    quantity: 1
    # vat_percent: 20
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
