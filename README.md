# invox

[![CI](https://github.com/0xboris/invox/actions/workflows/ci.yml/badge.svg)](https://github.com/0xboris/invox/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

invox turns invoice data in YAML into LaTeX and PDF invoices.

You keep your company, your customers and each invoice in plain YAML files. invox numbers new invoices, validates them, renders them into a LaTeX template, builds the PDF with [Tectonic](https://tectonic-typesetting.github.io), drafts the email to the customer and moves the finished invoice into an archive.

- Invoice numbers follow a pattern per customer, such as `CUST-001-007`, and never repeat a number already in the archive.
- Each line item can override the invoice's VAT rate, and the totals show one VAT line per rate.
- The starter template prints an EPC QR code, a SEPA transfer the customer can scan, on EUR invoices with an IBAN in the SEPA area.
- On macOS, `invox email` opens an Apple Mail draft with the PDF attached. Elsewhere it writes an `.eml` draft and opens it.
- stdout carries only data and `--json` prints JSON, so scripts can drive every command.

## Install

Install with Go 1.24 or later:

```sh
go install github.com/0xboris/invox/cmd/invox@latest
```

Homebrew and prebuilt binaries arrive with the first tagged release. These are the intended methods:

- Homebrew on macOS or Linux: `brew install 0xboris/tap/invox`.
- Release archives for Linux, macOS and Windows on amd64 and arm64, from the [releases page](https://github.com/0xboris/invox/releases). Each archive holds the binary, the man pages and completion scripts for bash, zsh, fish and PowerShell.

To build from a checkout, run `make build` (the binary is `./bin/invox`) or `make install`.

### Install Tectonic

`invox build` needs `tectonic` in your `PATH`. The other commands work without it. These commands come from the [Tectonic installation guide](https://tectonic-typesetting.github.io/book/latest/installation/), which lists more options.

On macOS:

```sh
brew install tectonic
```

On Linux or macOS, the download script puts `tectonic` in the current directory. Move it into a directory in your `PATH` afterwards.

```sh
curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net |sh
```

On Arch Linux:

```sh
sudo pacman -S tectonic
```

With conda, on any OS:

```sh
conda install -c conda-forge tectonic
```

On Windows, run the download script in PowerShell. It unpacks `tectonic.exe` in the current directory. Move it into a directory in your `PATH` afterwards.

```powershell
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://drop-ps1.fullyjustified.net'))
```

Tectonic downloads the LaTeX packages it needs on the first build, so the first `invox build` needs network access.

## Quick start

Create the config directory with starter files:

```sh
invox init
```

Put your company and payment details into `issuer.yaml`, in the config directory that `invox init` printed. Then add your customers. This command opens `customers.yaml` in your editor (`VISUAL`, then `EDITOR`):

```sh
invox customer edit
```

To change the numbering pattern, the archive directory or the email templates, run `invox config edit`, which opens `config.yaml`.

Create an invoice for a customer, check it and build the PDF:

```sh
invox new CUST-001 -e
invox validate CUST-001-001.yaml
invox build CUST-001-001.yaml
```

`new` writes `CUST-001-001.yaml` with the next free number and, with `-e`, opens it in your editor. `build` writes `CUST-001-001.pdf` next to it and sets the invoice status to `built`.

Draft the email, then archive the invoice:

```sh
invox email CUST-001-001.pdf
invox archive add CUST-001-001.yaml
```

Next month, start from the customer's last archived invoice. invox gives the copy the next number and fresh dates:

```sh
invox new CUST-001 --from-last
```

To fix an archived invoice, copy it out with `archive edit`, edit the copy in your editor, then rebuild and archive it again. invox asks before it replaces the archived file and keeps the old version in `.history/` inside the archive directory.

```sh
invox archive list
invox archive edit CUST-001-001.yaml
$EDITOR CUST-001-001.yaml
invox build CUST-001-001.yaml --archive
```

## Files

`invox init` writes these files to `$XDG_CONFIG_HOME/invox`, or `~/.config/invox` when `XDG_CONFIG_HOME` is unset. `INVOX_CONFIG_DIR` replaces that directory.

| File | Holds | Reference |
| --- | --- | --- |
| `config.yaml` | Path overrides, the numbering pattern, the archive directory and the email templates | `invox help config` |
| `customers.yaml` | One entry per customer: address, email, currency and numbering | `invox help customers` |
| `issuer.yaml` | Your company and payment details | `invox help issuer` |
| `invoice_defaults.yaml` | The document `invox new` copies into each new invoice | `invox help defaults` |
| `template.tex` | The LaTeX template with `@@PLACEHOLDER@@` tokens | `invox help template` |

An invoice file has the shape of `invoice_defaults.yaml`. `invox help defaults` describes it, and `invox validate` reports each wrong field with its file and line.

invox looks for `customers.yaml`, `issuer.yaml`, `invoice_defaults.yaml` and the template in this order:

1. The flag on the command line (`-c`, `-u`, `--defaults`, `-t`).
2. The current directory and its parents, up to the nearest directory with `.git`, `invox.yaml` or `invoice_defaults.yaml`.
3. `paths.*` in `config.yaml`.
4. The config directory.

So a project directory with its own `customers.yaml` overrides the global one. `invox config paths` prints the file each lookup finds. Archived invoices go to `archive.dir` from `config.yaml`, or by default to `~/.local/share/invox/invoices` on Linux, `~/Library/Application Support/invox/invoices` on macOS and `%APPDATA%\invox\invoices` on Windows. `invox help environment` lists the variables that move these directories.

## Documentation

`invox help` lists the commands and help topics. `invox help <command>` shows a command's flags and examples.

The same pages are in the repository:

- [`docs/cli/`](docs/cli/invox.md) has one Markdown page per command and help topic.
- [`share/man/man1/`](share/man/man1/) has the man pages. Read one with `man ./share/man/man1/invox-build.1`.

Both are generated from the command tree, so they match the binary.

The help topics are `config`, `customers`, `issuer`, `defaults`, `template`, `environment` and `exit-codes`. `invox help environment` lists the environment variables, the default directories on each OS and the order in which flags, `config.yaml` and defaults apply. `invox help exit-codes` lists the exit statuses.

Older command forms were removed and now fail with exit 2. Use the new forms:

| Removed | Use |
| --- | --- |
| `invox archive FILE` | `invox archive add FILE` |
| `invox customer config` | `invox customer edit` |
| `invox new -s FILE`, `--source FILE` | `invox new --defaults FILE` |
| `invox send` | `invox email` |
| Single-dash long flags such as `-input` | `--input` |

## Use invox in scripts

stdout carries only data. Status lines, hints, warnings, prompts and errors go to stderr, along with the output of `tectonic` and the editor.

- `new`, `increment`, `render`, `build`, `archive add` and `archive edit` print the path they wrote, one line, relative to the current directory when it is inside it. `email -o FILE` prints the draft path.
- `validate`, `init`, `config edit` and `customer edit` print nothing on stdout.
- `customer list`, `archive list` and `template list` print one tab-separated row per record. On a terminal they print a header and aligned columns instead.

```sh
pdf=$(invox build CUST-001-001.yaml)
invox email "$pdf" -o draft.eml
```

`--json <fields>` prints one JSON document on stdout. The list commands, `validate`, `new`, `increment`, `render`, `build`, `archive add` and `archive edit` take it, and each command's help lists its fields under "JSON fields".

```sh
invox validate CUST-001-001.yaml --json valid,total,currency
invox archive list --json file,customerId,number
```

These flags make runs safe to automate:

- `-n, --dry-run` runs the same checks as a real run, prints what it would do on stderr and writes nothing. `new`, `increment`, `render`, `build`, `email`, `archive add` and `archive edit` take it. Except for `email`, a dry run prints on stdout the same path the real run would print, so a script can read it first.
- `--force` lets `new`, `archive edit` and `email -o` overwrite an existing output file. On `new` and `archive edit` it never replaces a file inside the archive.
- `--yes` lets `archive add` and `build --archive` replace an archived invoice without asking.
- `--no-input`, or a non-empty `INVOX_PROMPT_DISABLED`, stops invox from prompting or opening an editor. A step that needs one fails with exit status 2.

Exit status 0 means success and 1 means the command failed. Exit status 2 is a usage error, a declined confirmation, or a confirmation or editor that was needed without a terminal. Exit statuses 130 and 143 mean Ctrl-C or SIGTERM stopped invox. See `invox help exit-codes`.

## Shell completion

`invox completion` prints a completion script for bash, zsh, fish or PowerShell. It completes commands, flags, customer IDs, template names and archived invoices. To load it in the current bash or zsh shell:

```sh
source <(invox completion bash)
source <(invox completion zsh)
```

For fish:

```sh
invox completion fish > ~/.config/fish/completions/invox.fish
```

For PowerShell:

```powershell
invox completion powershell | Out-String | Invoke-Expression
```

`invox completion --help` shows how to install each script permanently.

## Troubleshooting

`error: tectonic not found in PATH` means `invox build` can't find Tectonic. [Install Tectonic](#install-tectonic) and check that `tectonic --version` runs in the same shell.

`error: .../config.yaml:68: unknown key "patern" in numbering` means `config.yaml` has a typo or a value of the wrong type. invox reads `config.yaml` strictly and names the file and line. Run `invox config edit` to fix it. Invoice files, `customers.yaml`, `issuer.yaml` and `invoice_defaults.yaml` are strict in the same way.

If the PDF shows wrong or missing characters, check the font setup in your template. The starter template loads `fontspec` and sets `\tracinglostchars=3`, so a character the font can't show fails the build with an error instead of disappearing. `invox init` never overwrites an existing `template.tex`. If yours comes from an older version, replace its `\usepackage[T1]{fontenc}` and `\usepackage[utf8]{inputenc}` lines with `\usepackage{fontspec}` and `\tracinglostchars=3`. When you render outside the template's directory, invox copies the files the template references, such as `fonts/` or `logo.png`, next to the generated `.tex` file.

If an editor command exits 2 with a hint, stdin or stderr is not a terminal, or `--no-input` is set. Run the command in a terminal and keep stderr on it, without `2>`.

## Development

You need Go 1.24 or later. The Makefile wraps the checks that CI runs:

```sh
make build             # build ./bin/invox
make test              # go test -race ./...
make lint              # golangci-lint at the version CI uses
make vulncheck         # govulncheck ./...
make docs              # regenerate docs/cli and share/man/man1
make release-snapshot  # build every release archive into ./dist (needs GoReleaser v2)
make help              # list every target
```

`make fmt`, `make vet` and `make tidy` run gofmt, `go vet` and `go mod tidy -diff`. CI fails when `docs/cli` or `share/man/man1` don't match the command tree, so run `make docs` after you change any help text. The end-to-end tests in `cmd/invox/testdata/script/` pin the stdout, stderr and exit code of every command. After an intended output change, run `go test ./cmd/invox -run TestScript -update` and review the diff.

[`CLAUDE.md`](CLAUDE.md) describes the package layout and the test conventions. [`docs/design/`](docs/design/) holds design notes for shipped features.

On Windows, enable symlinks before you clone (`git config --global core.symlinks true`, which also needs Developer Mode or an administrator shell). Without them, `.claude/skills/quality-cli` checks out as a plain text file instead of a link to `.agents/skills/quality-cli`. The Go build and tests don't use it.

## Contributing

Bug reports and pull requests are welcome on [GitHub](https://github.com/0xboris/invox/issues). Keep each pull request to one issue and reference it with `Fixes #N`. A bug fix comes with a regression test that fails without the fix. stdout, stderr, exit codes, flag names and file formats are a contract, so change them only when the issue asks for it, and update the tests that pin them. [`CHANGELOG.md`](CHANGELOG.md) lists the user-visible changes.

## License

invox is released under the [MIT License](LICENSE).
