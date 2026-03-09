# Invox CLI

This directory contains the Go implementation of the invoice pipeline.

Run it directly:

```sh
cd go
go run ./cmd/invox validate -i invoice.yaml
go run ./cmd/invox render -i invoice.yaml
go run ./cmd/invox build -i invoice.yaml
go run ./cmd/invox customer list
go run ./cmd/invox config
go run ./cmd/invox customer config
go run ./cmd/invox new CUST-001
go run ./cmd/invox increment -i invoice.yaml
```

Build a local binary:

```sh
cd go
go build -o invox ./cmd/invox
./invox validate -i invoice.yaml
./invox render -i invoice.yaml
./invox build -i invoice.yaml
./invox customer list
./invox config
./invox customer config
./invox new CUST-001
./invox increment -i invoice.yaml
```

Install it from this repository checkout:

```sh
cd go
go install ./cmd/invox
```

Help:

```sh
go run ./cmd/invox -h
go run ./cmd/invox help config
go run ./cmd/invox config -h
go run ./cmd/invox customer -h
go run ./cmd/invox customer config -h
go run ./cmd/invox render -h
go run ./cmd/invox build -h
```

Flag aliases:

- `-i, --input` for the invoice YAML file
- `-o, --output` for the render/build output path
- `-c, --customers` for `customers.yaml`
- `-u, --issuer` for `issuer.yaml`
- `-s, --source` for `invoice_defaults.yaml` on `new`
- `-t, --template` for the LaTeX template

Customer commands:

- `customer list` prints one customer per line as `CUSTOMER_ID<TAB>NAME<TAB>STATUS`
- `customer config` opens the resolved `customers.yaml` file in the default shell editor
- Preferred customer fields are `name`, `email`, `address.*`, and `tax.vat_tax_id`
- `billing.currency` is optional and defaults to `EUR`
- Legacy aliases `legal_company_name` and `billing.email` are still accepted

Other commands:

- `config` opens the resolved `config.yaml` file in the default shell editor
- if `config.yaml` does not exist yet, `config` creates it with a commented template
- `help config` prints the supported `config.yaml` keys and the commented template without modifying the file

Defaults:

- Global config directory: `$XDG_CONFIG_HOME/invox` or `~/.config/invox`
- The CLI prefers `invox`, but still falls back to the legacy `invoice-tool` config directory if it already exists.
- Global default files:
  - `config.yaml`
  - `customers.yaml`
  - `issuer.yaml`
  - `invoice_defaults.yaml`
  - `template.tex`
- Archive markdown directory:
  - defaults to `~/Library/Application Support/invox/invoices` on macOS
  - can be overridden in `config.yaml` via `archive.dir`
  - resolution is: `config.yaml`, then platform default
- `config.yaml` currently supports:
  - `paths.customers`
  - `paths.issuer`
  - `paths.defaults`
  - `paths.template`
  - `numbering.pattern`
  - `numbering.start` as the global fallback start
  - `archive.dir`
- `customers.yaml` can override the global numbering start per customer via `<customer>.numbering.start`
- A freshly created `config.yaml` is seeded with a commented template like:

```yaml
# paths:
#   customers: 'customers.yaml'
#   issuer: 'issuer.yaml'
#   defaults: 'invoice_defaults.yaml'
#   template: 'template.tex'
#
# numbering:
#   pattern: '{customer_code}-{counter:03}'
#   start: 1
#
# archive:
#   dir: '~/Library/Application Support/invox/invoices'
```
- Resolution order for support files is: explicit flag, upward project search, `paths.*` from `config.yaml`, then global config files.
- `-i, --input` is still required.
- `invoice.number` is required for `validate`, `render`, and `build`.
- `new` defaults to `./invoice.yaml` if `-o` is omitted.
- `render` defaults to `./invoice.tex` if `-o` is omitted.
- `build` defaults to `./invoice.pdf` if `-o` is omitted.
- `build` renders in a temporary directory and leaves only the final PDF at the output path.
- When rendering outside the template directory, the CLI copies referenced assets like `fonts/` and `logo.png` next to the generated TeX so Tectonic can build successfully.
- `new` creates a fresh invoice YAML from `invoice_defaults.yaml` and derives the next number by scanning archived invoices.
- `new` calculates and prefills `invoice.due_date` from `issuer.yaml -> payment.due_days`.
- `increment` updates `invoice.number` in place using the larger of the current invoice counter and the archived-invoice counter.
- `payment_terms_text` now comes from `issuer.yaml -> payment.payment_terms_text`, not from invoice input files.
- `due_days` now comes from `issuer.yaml -> payment.due_days` and must be a non-negative integer day count.

Numbering config in `config.yaml`:

```yaml
numbering:
  pattern: "{customer_code}-{counter:03}"
  start: 1
```

Per-customer override in `customers.yaml`:

```yaml
CUST-001:
  name: Appsters GmbH
  numbering:
    code: APP
    start: 100
```

Customer entry example:

```yaml
0021:
  status: active
  name: Appsters GmbH
  email: office@appsters.at
  address:
    street: Griesgasse 19
    postal_code: "9020"
    city: Klagenfurt
    country: Oesterreich
  tax:
    vat_tax_id: ATU80037005
  billing:
    currency: EUR
```

`billing.currency` is optional and defaults to `EUR`.

Supported pattern tokens:

- `{customer_id}` from `invoice.customer_id`
- `{customer_code}` from `customers.yaml -> <customer>.numbering.code`, falling back to `customer_id`
- `{year}`, `{month}`, `{day}` from `invoice.issue_date`
- `{counter}` or `{counter:03}` for the per-customer counter

The pattern must include a customer token and a counter token.
If `customers.yaml -> <customer>.numbering.start` is set, it overrides `config.yaml -> numbering.start` for that customer.

Examples with explicit overrides:

```sh
go run ./cmd/invox new CUST-001 -o invoices/2026-0022.yaml -s invoice_defaults.yaml -c customers.yaml -u issuer.yaml
go run ./cmd/invox increment -i invoices/2026-0022.yaml -c customers.yaml
go run ./cmd/invox customer list -c customers.yaml
go run ./cmd/invox customer config -c customers.yaml
go run ./cmd/invox validate -i invoice.yaml -c customers.yaml -u issuer.yaml
go run ./cmd/invox render -i invoice.yaml -o out/invoice.tex -c customers.yaml -u issuer.yaml -t invoice_template.tex
go run ./cmd/invox build -i invoice.yaml -o out/invoice.pdf -c customers.yaml -u issuer.yaml -t invoice_template.tex
```

Example using global defaults only:

```sh
mkdir -p ~/.config/invox
cp customers.yaml ~/.config/invox/customers.yaml
cp issuer.yaml ~/.config/invox/issuer.yaml
cp invoice_defaults.yaml ~/.config/invox/invoice_defaults.yaml
cp invoice_template.tex ~/.config/invox/template.tex
cp logo.png ~/.config/invox/logo.png
cp -R fonts ~/.config/invox/fonts

cat > ~/.config/invox/config.yaml <<'EOF'
numbering:
  pattern: "{customer_code}-{counter:03}"
  start: 1
archive:
  dir: ~/Documents/Invox/Invoices
EOF

go run ./cmd/invox new CUST-001
go run ./cmd/invox increment -i invoice.yaml
go run ./cmd/invox customer list
go run ./cmd/invox config
go run ./cmd/invox customer config
go run ./cmd/invox validate -i invoice.yaml
go run ./cmd/invox render -i invoice.yaml
go run ./cmd/invox build -i invoice.yaml
```
