# Invox CLI

`invox` is a command-line invoice workflow tool. It creates invoice YAML files, validates them against customer and issuer data, renders LaTeX, builds PDFs, and manages the invoice lifecycle from draft to built to archived, including reopening archived invoices for editing.

Run it directly:

```sh
go run ./cmd/invox validate -i invoice.yaml
go run ./cmd/invox render -i invoice.yaml
go run ./cmd/invox email invoice.pdf
go run ./cmd/invox build invoice.yaml
go run ./cmd/invox customer list
go run ./cmd/invox config
go run ./cmd/invox customer config
go run ./cmd/invox init
go run ./cmd/invox new CUST-001
go run ./cmd/invox new CUST-001 --from-last
go run ./cmd/invox increment -i invoice.yaml
go run ./cmd/invox archive invoice.yaml
go run ./cmd/invox archive edit 2026-03-06.yaml
go run ./cmd/invox archive list
```

Build a local binary:

```sh
go build -o invox ./cmd/invox
./invox validate -i invoice.yaml
./invox render -i invoice.yaml
./invox email invoice.pdf
./invox build invoice.yaml
./invox customer list
./invox config
./invox customer config
./invox init
./invox new CUST-001
./invox new CUST-001 --from-last
./invox increment -i invoice.yaml
./invox archive invoice.yaml
./invox archive edit 2026-03-06.yaml
./invox archive list
```

Install it from this repository checkout:

```sh
go install ./cmd/invox
invox init
```

Make shortcuts for contributors:

```sh
make build
make test
make install
make init
make validate
make render
make email
make pdf
make archive
```

The Makefile is only contributor convenience around the real CLI. The user-facing interface remains `invox ...`.

- `make build` builds `./bin/invox`
- `make install` runs `go install ./cmd/invox`
- `make init` runs `./bin/invox init`
- `make validate`, `make render`, `make email`, `make pdf`, and `make archive` run the local `./bin/invox` by default
- those targets accept `INPUT`, `PDF_INPUT`, `CUSTOMERS`, `ISSUER`, `TEMPLATE`, `TEX_OUTPUT`, `PDF_OUTPUT`, `EMAIL_OUTPUT`, `EMAIL_TO`, `EMAIL_SUBJECT`, `CLI`, and `ARGS`
- `PDF_OUTPUT` defaults to the `INPUT` path with a `.pdf` extension
- `EMAIL_OUTPUT` defaults to the `INPUT` path with a `.eml` extension
- if `invox` is already installed and on `PATH`, you can use `CLI=invox`

Examples:

```sh
make render CUSTOMERS=customers.yaml ISSUER=issuer.yaml TEMPLATE=invoice_template.tex
make email INPUT=invoice.yaml
make pdf CLI=invox INPUT=invoice.yaml
make archive CLI=invox INPUT=invoice.yaml
```

Help:

```sh
go run ./cmd/invox -h
go run ./cmd/invox help config
go run ./cmd/invox config -h
go run ./cmd/invox init -h
go run ./cmd/invox customer -h
go run ./cmd/invox customer config -h
go run ./cmd/invox render -h
go run ./cmd/invox email -h
go run ./cmd/invox build -h
go run ./cmd/invox archive -h
```

Flag aliases:

- `-i, --input` for the invoice YAML file, or the built PDF on `email`
- `-o, --output` for the render/build output path
- `-o, --output` for the render/build/email output path
- `-c, --customers` for `customers.yaml`
- `-u, --issuer` for `issuer.yaml`
- `-p, --pdf` for the built invoice PDF on `email`
- `-s, --source` for `invoice_defaults.yaml` on `new`
- `--from-last` for cloning the latest archived invoice of the requested customer on `new`
- `--to` for overriding the recipient email on `email`
- `--subject` for overriding the draft email subject on `email`; placeholder expansion is supported there too
- `-t, --template` for the LaTeX template
- `email`, `build`, and `archive` also accept the invoice path positionally, for example `invox email invoice.pdf` or `invox build invoice.yaml`
- `build` also supports `--archive` to archive the invoice after a successful PDF build
- `archive edit` copies an archived invoice into the current directory as YAML with `invoice.status: editing`
- `archive list` prints one archived invoice per line as `FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS`

Customer commands:

- `customer list` prints one customer per line as `CUSTOMER_ID<TAB>NAME<TAB>STATUS`
- `customer config` opens the resolved `customers.yaml` file in the default shell editor
- Preferred customer fields are `name`, `email_greeting`, `contact_person`, `email`, `address.*`, and `tax.vat_tax_id`
- `billing.currency` is optional and defaults to `EUR`
- Legacy aliases `legal_company_name` and `billing.email` are still accepted

Other commands:

- `init` creates starter versions of `config.yaml`, `customers.yaml`, `issuer.yaml`, `invoice_defaults.yaml`, and `template.tex` in the global config directory
- existing non-empty support files are left unchanged by `init`
- `template list` prints available templates from the same directory as the resolved default template as `TEMPLATE_NAME<TAB>ABSOLUTE_PATH`
- `template list --names` prints just template names, one per line
- `completion zsh` prints a Zsh completion script with template-name completion for `render` and `build`
- `config` opens the resolved `config.yaml` file in the default shell editor
- if `config.yaml` does not exist yet, `config` creates it with a commented template
- `help config` prints the supported `config.yaml` keys and the commented template without modifying the file
- `email` opens an editable compose window in Apple Mail on macOS; if `-o` is set, or on other platforms, it creates a `.eml` draft with the invoice PDF attached, opens it, and then schedules the `.eml` file for cleanup shortly after
- `email` accepts either the invoice YAML or the built PDF as input; when given the PDF, it looks for the same-basename YAML next to the PDF first, then in `archive.dir`

Defaults:

- Global config directory: `$XDG_CONFIG_HOME/invox` or `~/.config/invox`
- The CLI prefers `invox`, but still falls back to the legacy `invoice-tool` config directory if it already exists.
- Global default files:
  - `config.yaml`
  - `customers.yaml`
  - `issuer.yaml`
  - `invoice_defaults.yaml`
  - `template.tex`
- Archive directory:
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
  - `email.subject`
  - `email.body`
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
- `-t, --template` accepts either a path or a known template filename such as `multi_vat.tex`; filename-only lookup uses the same directory as the resolved default template.
- `-i, --input` is required for `increment`, `validate`, and `render`.
- `invoice.number` is required for `validate`, `render`, and `build`.
- `new` defaults to `./<invoice.number>.yaml` if `-o` is omitted.
- `new --from-last` uses the latest archived invoice for that customer as the source document instead of `invoice_defaults.yaml`.
- `render` defaults to `./invoice.tex` if `-o` is omitted.
- `email` defaults to the input path with a `.eml` extension if `-o` is omitted.
- `build` defaults to the input path with a `.pdf` extension if `-o` is omitted.
- `build` renders in a temporary directory and leaves only the final PDF at the output path.
- `email` defaults to the input path with a `.pdf` extension for the attachment if `-p` is omitted, or uses the input file itself when the input is already a PDF.
- when `email` gets a PDF input, it resolves the invoice YAML from the same directory first and then falls back to `archive.dir`
- `email` requires `invoice.status: built` or `archived`, writes an `X-Unsent: 1` email draft, opens it in the default mail app, schedules the `.eml` file for cleanup shortly after, and leaves the invoice status unchanged.
- `email.subject` and `email.body` in `config.yaml` can override the draft subject and plain-text body. Both support `{customer_name}`, `{email_greeting}`, `{contact_person}`, `{customer_id}`, `{invoice_number}`, `{issue_date}`, `{due_date}`, `{total_amount}`, `{outstanding_amount}`, `{payment_terms_text}`, and `{issuer_name}`.
- `build` updates `invoice.status` to `built` after a successful PDF build.
- `build --archive` archives the invoice immediately after a successful PDF build.
- `archive` moves a built invoice into `archive.dir`, or replaces an archived invoice when the working copy came from `archive edit`.
- `archive edit <FILENAME>` uses the relative filename from `archive list`, copies that archived invoice into the current directory, and marks the working copy as `editing`.
- Re-archiving a working copy created by `archive edit` replaces the archived invoice and rewrites the status back to `archived`.
- When rendering outside the template directory, the CLI copies referenced assets like `fonts/` and `logo.png` next to the generated TeX so Tectonic can build successfully.
- `new` creates a fresh invoice YAML from `invoice_defaults.yaml` and derives the next number by scanning archived invoices.
- `new --from-last` keeps the previous invoice content for that customer but regenerates `invoice.number`, `invoice.issue_date`, `invoice.due_date`, `invoice.status`, and `invoice.paid_amount`.
- `new` calculates and prefills `invoice.due_date` from `issuer.yaml -> payment.due_days`.
- `increment` updates `invoice.number` in place using the larger of the current invoice counter and the archived-invoice counter.
- `payment_terms_text` now comes from `issuer.yaml -> payment.payment_terms_text`, not from invoice input files.
- `due_days` now comes from `issuer.yaml -> payment.due_days` and must be a non-negative integer day count.
- `vat_label` can be set at `issuer.yaml -> payment.vat_label` to override the label used by `@@VAT_SUMMARY_ROWS@@`. It defaults to `VAT`.
- `issuer.yaml -> payment.epc_qr.*` can customize the EPC payment QR output emitted by `@@EPC_QR_LABEL@@` and `@@EPC_QR_CODE@@`.
  Supported keys are `label`, `name`, `purpose`, `text`, and `information`.
  `label` defaults to `Pay via EPC-QR`, `text` defaults to `invoice.number`, `name` defaults to `issuer.company.legal_company_name`, and the encoded amount uses the invoice outstanding amount.
  EPC QR generation also requires `issuer.payment.iban` to be a valid IBAN within the current SEPA scheme scope.
  The starter/default template uses `@@EPC_QR_AVAILABLE@@`, `@@EPC_QR_LABEL@@`, and `@@EPC_QR_CODE@@`, and the label/code stay empty unless a real QR code can be rendered.

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
  email_greeting: Dear Jane Doe,
  contact_person: Jane Doe
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

## Issuer Configuration

`issuer.yaml` stores your own company details and payment defaults.

You can print the same reference in the CLI with:

```sh
invox help issuer
```

`invox init` also creates a starter `issuer.yaml` in the global config directory.

Supported shape:

```yaml
company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: "1010"
    city: Vienna
    country: Austria

payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
  vat_label: VAT
  epc_qr:
    label: Pay via EPC-QR
    purpose: SUPP
    information: Scan to pay this invoice
    # name: Boris Consulting
    # text: 2026-0001
```

Required fields:

- `company.legal_company_name`
- `company.company_registration_number`
- `company.vat_tax_id`
- `company.website`
- `company.email`
- `company.address.street`
- `company.address.postal_code`
- `company.address.city`
- `company.address.country`
- `payment.bank_name`
- `payment.iban`
- `payment.bic`
- `payment.due_days`
- `payment.payment_terms_text`

Optional fields:

- `payment.vat_label` overrides the VAT label used by `@@VAT_SUMMARY_ROWS@@`; default is `VAT`
- `payment.epc_qr.label` overrides the QR label; default is `Pay via EPC-QR`
- `payment.epc_qr.name` overrides the QR recipient name; default is `company.legal_company_name`
- `payment.epc_qr.purpose` sets the EPC QR purpose code
- `payment.epc_qr.text` overrides the QR text line; default is `invoice.number`
- `payment.epc_qr.information` sets EPC QR remittance information

Rules:

- `payment.due_days` must be a non-negative integer
- `new` uses `payment.due_days` to prefill `invoice.due_date`
- templates and email placeholders use `payment.payment_terms_text`
- EPC QR output requires `payment.iban` to be a valid IBAN within the current SEPA scheme scope

## Template Authoring

`invox render` and `invox build` read a LaTeX template file, replace known placeholders, and write the rendered `.tex` before optionally compiling it to PDF.

Basic workflow:

1. Put your template in `template.tex`, keep it in the config directory, or pass it explicitly with `-t, --template`.
2. Use literal placeholder tokens like `@@INVOICE_NUMBER@@` directly in the LaTeX source.
3. Run `invox render invoice.yaml` to inspect the generated `.tex`.
4. Run `invox build invoice.yaml` once the rendered LaTeX looks correct.

Template discovery:

- `invox template list` shows `.tex` files from the same directory as the resolved default template.
- `-t, --template` accepts either a full path or a listed filename like `multi_vat.tex`.
- `invox completion zsh` emits a Zsh completion script that completes template names for `render` and `build`.

Shell completion:

- Temporary in the current shell: `source <(invox completion zsh)`
- Persistent Zsh install:

```zsh
mkdir -p ~/.zsh/completions
invox completion zsh > ~/.zsh/completions/_invox
```

Add this before `compinit` in `~/.zshrc`:

```zsh
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit
compinit
```

Important rules:

- Placeholder names are case-sensitive and must match exactly.
- Unknown placeholders are left unchanged in the rendered TeX.
- `@@VAT_RATE@@` and `@@VAT_AMOUNT@@` are no longer supported; use `@@VAT_SUMMARY_ROWS@@`.
- Most placeholders are LaTeX-escaped automatically, so customer names, addresses, and free text are safe to insert directly.
- `@@LINE_ITEMS_ROWS@@` is different: it expands to multiple LaTeX table rows and must be placed inside a table environment with five columns.
- `@@LINE_ITEMS_ROWS_WITH_VAT@@` expands to multiple LaTeX table rows and must be placed inside a table environment with six columns.
- Dates from YAML like `2026-03-10` are rendered as `10.03.2026`.
- Money values are rendered like `1.234,56 \euro` for EUR, or `1.234,56 USD` for other currencies.

Minimal example:

```tex
\documentclass{article}
\usepackage{longtable}
\usepackage{eurosym}
\begin{document}

Invoice @@INVOICE_NUMBER@@

\textbf{Bill To}\\
@@CUSTOMER_NAME@@\\
@@CUSTOMER_STREET@@\\
@@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@\\
@@CUSTOMER_COUNTRY@@

\begin{longtable}{p{3cm}p{5.5cm}r r r}
Item & Description & Unit Price & Qty & Total\\
@@LINE_ITEMS_ROWS@@
\end{longtable}

Total: @@TOTAL@@

\end{document}
```

### Available Placeholders

Issuer:

- `@@ISSUER_NAME@@`
- `@@ISSUER_COMPANY_REG_NO@@`
- `@@ISSUER_VAT_TAX_ID@@`
- `@@ISSUER_WEBSITE@@`
- `@@ISSUER_EMAIL@@`
- `@@ISSUER_STREET@@`
- `@@ISSUER_CITY@@`
- `@@ISSUER_POSTAL_CODE@@`
- `@@ISSUER_COUNTRY@@`

Customer:

- `@@CUSTOMER_NAME@@`
- `@@CUSTOMER_STREET@@`
- `@@CUSTOMER_CITY@@`
- `@@CUSTOMER_POSTAL_CODE@@`
- `@@CUSTOMER_COUNTRY@@`
- `@@CUSTOMER_VAT_TAX_ID@@`
- `@@CUSTOMER_EMAIL@@`

Invoice metadata:

- `@@INVOICE_NUMBER@@`
- `@@ISSUE_DATE@@`
- `@@DUE_DATE@@`
- `@@PERIOD_LABEL@@`

Line items:

- `@@LINE_ITEMS_ROWS@@`
  Expands to all invoice positions as LaTeX rows in this order:
  `name`, `description`, `unit price`, `quantity`, `line total`
- `@@LINE_ITEMS_ROWS_WITH_VAT@@`
  Expands to all invoice positions as LaTeX rows in this order:
  `name`, `description`, `unit price`, `quantity`, `VAT rate`, `line total`

Totals:

- `@@SUBTOTAL@@`
- `@@VAT_SUMMARY_ROWS@@`
  Expands to one or more LaTeX rows like `VAT (20\%): & 40,00 \euro\\`
  The row label defaults to `VAT` and can be overridden with `issuer.yaml -> payment.vat_label`.
- `@@TOTAL@@`
- `@@PAID_AMOUNT@@`
- `@@OUTSTANDING_AMOUNT@@`
- `@@INVOICE_TOTAL@@`
  Alias for the invoice total amount
- `@@OUTSTANDING_TOTAL@@`
  Alias for the outstanding amount

Payment:

- `@@PAYMENT_TERMS_TEXT@@`
- `@@VAT_LABEL@@`
- `@@BANK_NAME@@`
- `@@IBAN@@`
- `@@BIC@@`
- `@@EPC_QR_AVAILABLE@@`
  Expands to `1` when `@@EPC_QR_CODE@@` is active and a valid EPC QR code can be rendered, otherwise `0`.
- `@@EPC_QR_LABEL@@`
  Expands to the plain EPC QR label text, or to an empty string when `@@EPC_QR_CODE@@` is inactive or cannot be rendered.
- `@@EPC_QR_CODE@@`
  Expands to the low-level `\qrcode{...}` command carrying an EPC069-12 SEPA credit-transfer payload.
  It expands to an empty string when EPC QR is not eligible and otherwise validates the eligible EPC data.

### Using `@@LINE_ITEMS_ROWS@@`

`@@LINE_ITEMS_ROWS@@`, `@@LINE_ITEMS_ROWS_WITH_VAT@@`, and `@@VAT_SUMMARY_ROWS@@` render structured LaTeX instead of a single text value.

It currently emits rows for a five-column table:

```tex
\begin{longtable}{p{3cm}p{5.5cm}r r r}
\toprule
Item & Description & Unit Price & Qty & Total\\
\midrule
\endhead
@@LINE_ITEMS_ROWS@@
\end{longtable}
```

If you change the table to fewer or more columns, you also need to change how rows are generated in the Go code.

For a VAT-aware table, use `@@LINE_ITEMS_ROWS_WITH_VAT@@` in a six-column layout:

```tex
\begin{longtable}{p{3cm}p{5.5cm}r r r r}
\toprule
Item & Description & Unit Price & Qty & @@VAT_LABEL@@ & Total\\
\midrule
\endhead
@@LINE_ITEMS_ROWS_WITH_VAT@@
\end{longtable}
```

For totals, use `@@VAT_SUMMARY_ROWS@@` inside a two-column table:

```tex
\begin{tabular}{lr}
Subtotal: & @@SUBTOTAL@@\\
@@VAT_SUMMARY_ROWS@@
Total: & @@TOTAL@@\\
\end{tabular}
```

If you want the same label elsewhere in the template, reuse `@@VAT_LABEL@@`.

### Using `@@EPC_QR_AVAILABLE@@`, `@@EPC_QR_LABEL@@`, and `@@EPC_QR_CODE@@`

`@@EPC_QR_AVAILABLE@@`, `@@EPC_QR_LABEL@@`, and `@@EPC_QR_CODE@@` are the recommended EPC QR placeholders for starter/default templates.

Add the package and optional size setting in the template preamble:

```tex
\usepackage{qrcode}
\qrset{height=2.2cm}
```

Place the placeholders where the EPC payment panel should appear:

```tex
\begin{tabular}{lr}
Subtotal: & @@SUBTOTAL@@\\
@@VAT_SUMMARY_ROWS@@
Total: & @@TOTAL@@\\
Paid: & @@PAID_AMOUNT@@\\
Outstanding: & @@OUTSTANDING_AMOUNT@@\\
\end{tabular}
\ifnum@@EPC_QR_AVAILABLE@@=1
  \vspace{0.5cm}
  {\bfseries @@EPC_QR_LABEL@@\par}
  @@EPC_QR_CODE@@
\fi
```

Rules:

- The payload follows EPC069-12 for SCT invoice-style QR codes.
- `@@EPC_QR_AVAILABLE@@` expands to `0` for non-EUR invoices.
- `@@EPC_QR_AVAILABLE@@` expands to `0` when the outstanding amount is `0` or below.
- EPC QR generation requires the beneficiary IBAN to be within the current SEPA scheme scope.
- `@@EPC_QR_LABEL@@` only renders when `@@EPC_QR_CODE@@` is also active and a valid EPC QR payload can be produced.
- The label defaults to `Pay via EPC-QR` and can be overridden with `issuer.yaml -> payment.epc_qr.label`.
- Size is controlled in the template with `\qrset{height=...}`.
- EPC placeholders are resolved with plain text replacement, not TeX-aware parsing.
- Do not place `@@EPC_QR_AVAILABLE@@`, `@@EPC_QR_LABEL@@`, or `@@EPC_QR_CODE@@` inside comments, verbatim/listing environments, or literal template examples unless you intend them to be treated as active placeholders.

For custom templates, you can also use `@@EPC_QR_CODE@@` on its own when you do not need a label or a template-side conditional.

- The QR code encodes the outstanding amount, not the invoice total.
- The QR payload is encoded as real UTF-8 bytes.
- Supported overrides in `issuer.yaml -> payment.epc_qr` are:
  - `label`
  - `name`
  - `purpose`
  - `text`
  - `information`

### Assets and Relative Paths

If your template references local assets and you render into another directory, `invox` copies supported relative assets next to the generated TeX automatically.

Supported patterns:

- `\includegraphics{logo.png}`
- font directory references like `Path=fonts/`

Rules:

- Relative paths work best
- Absolute paths are not copied
- Parent-directory paths like `../assets/logo.png` are not copied
- If you keep your template, fonts, and images together, `render` and `build` work more predictably

### Recommended Workflow

When creating or adjusting a template:

1. Start from the generated starter `template.tex`.
2. Edit the layout and placeholders there.
3. Run `invox render invoice.yaml -t template.tex`.
4. Inspect the generated `.tex` if something looks wrong.
5. Run `invox build invoice.yaml -t template.tex` once the TeX renders correctly.

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
go run ./cmd/invox new CUST-001 --from-last -c customers.yaml -u issuer.yaml
go run ./cmd/invox increment -i invoices/2026-0022.yaml -c customers.yaml
go run ./cmd/invox customer list -c customers.yaml
go run ./cmd/invox customer config -c customers.yaml
go run ./cmd/invox template list
go run ./cmd/invox completion zsh
go run ./cmd/invox validate -i invoice.yaml -c customers.yaml -u issuer.yaml
go run ./cmd/invox render -i invoice.yaml -o out/invoice.tex -c customers.yaml -u issuer.yaml -t multi_vat.tex
go run ./cmd/invox email invoice.yaml -c customers.yaml -u issuer.yaml
go run ./cmd/invox email invoice.pdf -c customers.yaml -u issuer.yaml
go run ./cmd/invox email invoice.yaml --to billing@example.com --subject "Invoice {invoice_number} for {customer_name}" -c customers.yaml -u issuer.yaml
go run ./cmd/invox build invoice.yaml -o out/invoice.pdf -c customers.yaml -u issuer.yaml -t multi_vat.tex
go run ./cmd/invox build invoice.yaml --archive -c customers.yaml -u issuer.yaml -t multi_vat.tex
go run ./cmd/invox archive invoice.yaml
go run ./cmd/invox archive edit 2026-03-06.yaml
go run ./cmd/invox archive list
```

Example using global defaults only:

```sh
go run ./cmd/invox init
go run ./cmd/invox new CUST-001
go run ./cmd/invox increment -i invoice.yaml
go run ./cmd/invox customer list
go run ./cmd/invox config
go run ./cmd/invox customer config
go run ./cmd/invox validate -i invoice.yaml
go run ./cmd/invox render -i invoice.yaml
go run ./cmd/invox email invoice.pdf
go run ./cmd/invox build invoice.yaml --archive
go run ./cmd/invox archive edit 2026-03-06.yaml
go run ./cmd/invox archive 2026-03-06.yaml
go run ./cmd/invox archive list
```
