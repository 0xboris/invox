# invox

`invox` is a Go CLI for YAML- and LaTeX-driven invoice workflows. It creates invoice drafts, validates them against customer and issuer data, renders LaTeX, builds PDFs with Tectonic, drafts invoice emails, and archives finished invoices for reuse.

## What It Covers

- Bootstraps a working invoice setup with starter files
- Generates invoice numbers from customer-aware numbering rules and archive history
- Validates invoice YAML before render, build, or email
- Renders `.tex` invoices from plain LaTeX templates with `@@PLACEHOLDER@@` tokens
- Builds PDFs with `tectonic`
- Drafts invoice emails from either invoice YAML or an already-built PDF
- Archives invoices and supports reopening archived invoices for editing

## Requirements

- Go `1.22+`
- `tectonic` in `PATH` for `invox build`
- A shell editor configured via `VISUAL` or `EDITOR` for `invox config`, `invox customer config`, and `invox new -e`

Install `tectonic` on macOS with:

```sh
brew install tectonic
```

## Installation

Install from this checkout:

```sh
go install ./cmd/invox
```

Or build a local binary:

```sh
go build -o ./bin/invox ./cmd/invox
./bin/invox -h
```

## Quick Start

Initialize the global config directory and starter files:

```sh
invox init
```

Edit the generated files:

```sh
invox config
invox customer config
```

Create, validate, build, email, and archive an invoice:

```sh
invox new CUST-001 -o invoice.yaml -e
invox validate -i invoice.yaml
invox render -i invoice.yaml -o invoice.tex
invox build invoice.yaml
invox email invoice.pdf
invox archive invoice.yaml
```

Common shortcuts:

```sh
invox new CUST-001 --from-last
invox build invoice.yaml --archive
invox archive edit 2026-03-06.yaml
invox archive list
```

## Default Files

`invox init` creates these starter files in `$XDG_CONFIG_HOME/invox` or `~/.config/invox`:

| File | Purpose |
| --- | --- |
| `config.yaml` | Global path overrides, numbering, archive settings, and email templates |
| `customers.yaml` | Customer records, billing defaults, and per-customer numbering |
| `issuer.yaml` | Your company and payment details |
| `invoice_defaults.yaml` | Source document used by `invox new` |
| `template.tex` | Default LaTeX invoice template |

Support files resolve in this order:

1. Explicit CLI flags
2. Upward search from the current project directory
3. `paths.*` overrides in `config.yaml`
4. Conventional files in the global config directory

## Command Overview

| Command | Purpose |
| --- | --- |
| `invox init` | Create starter support files in the global config directory |
| `invox config` | Open `config.yaml` in the default shell editor |
| `invox customer list` | List customers from `customers.yaml` |
| `invox customer config` | Open `customers.yaml` in the default shell editor |
| `invox template list` | List available LaTeX templates |
| `invox completion zsh` | Generate Zsh completion output |
| `invox new CUSTOMER_ID` | Create a new invoice with a generated number |
| `invox increment -i invoice.yaml` | Increment an existing invoice number in place |
| `invox validate -i invoice.yaml` | Validate invoice data against customer and issuer files |
| `invox render -i invoice.yaml` | Render a LaTeX invoice file |
| `invox build invoice.yaml` | Render and compile a PDF with `tectonic` |
| `invox email invoice.yaml` | Draft an email with the invoice PDF attached |
| `invox archive invoice.yaml` | Move a built or edited invoice into the archive |
| `invox archive edit FILENAME` | Copy an archived invoice into the current directory as an editable working copy |
| `invox archive list` | List archived invoices |

Run `invox -h` for the top-level command summary. The built-in help also includes focused references for the supported file formats and template system:

```sh
invox help config
invox help customers
invox help issuer
invox help defaults
invox help template
```

## Notes

- `invox new` writes `./<invoice.number>.yaml` by default. `--from-last` clones the latest archived invoice for the customer and refreshes numbering and dates.
- `invox build` writes to the input path with a `.pdf` extension by default, updates the invoice status to `built`, and can archive immediately via `--archive`.
- `invox email` accepts either the invoice YAML or a built PDF. When given a PDF, it looks for the matching YAML next to the PDF first and then in `archive.dir`.
- On macOS, `invox email` opens an editable Apple Mail compose window when possible. Otherwise it creates an `.eml` draft, opens it, and schedules that file for cleanup.
- The starter template already includes VAT summary support and EPC QR placeholders for eligible EUR invoices with a SEPA-scope IBAN.
- When rendering outside the template directory, `invox` copies referenced assets such as `fonts/` and `logo.png` next to the generated TeX so `tectonic` can build successfully.
- `invox` prefers the `invox` config directory but still falls back to the legacy `invoice-tool` directory when it already exists.

## Development

The `Makefile` is a thin convenience layer around the real CLI:

```sh
make build
make test
make vet
make install
make init
make validate
make render
make email
make pdf
make archive
```

For day-to-day usage, prefer the actual interface:

```sh
invox ...
```

Last reviewed: 2026-03-30
