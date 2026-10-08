# Changelog

All notable changes to invox are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **BREAKING:** stdout now carries only data. `new`, `increment`, `render`, `build`, `archive`, `archive edit` and `email -o` print only the path they created or changed, and `validate`, `init`, `config` and `customer config` print nothing on stdout. Status lines, hints, warnings, prompts and the output of tectonic, the editor and the opener go to stderr, so scripts that read status lines from stdout must read stderr or use the printed path. ([#80](https://github.com/0xboris/invox/pull/80))
- **BREAKING:** `validate` shows amounts as `120,00 €` instead of the LaTeX `120,00 \euro`, and other currency codes are no longer LaTeX-escaped (`US$` instead of `US\$`). ([#80](https://github.com/0xboris/invox/pull/80))
- **BREAKING:** usage errors exit 2 and print two lines, `error: <message>` and `Run 'invox <cmd> --help' for usage.`, instead of the full help page. Every error starts with `error:`, and paths under the current directory are shown relative to it. ([#77](https://github.com/0xboris/invox/pull/77))
- **BREAKING:** `invox email` with a PDF that has no matching invoice YAML exits 1 instead of 2, and a failed tectonic run reports `error: tectonic exited with status N`. ([#77](https://github.com/0xboris/invox/pull/77))
- **BREAKING:** the starter `template.tex` that `invox init` writes uses `fontspec` and `\tracinglostchars=3` instead of `fontenc` and `inputenc`. Accented names and addresses now appear correctly, and a character the font can't show fails `invox build`. The new template needs a fontspec-capable engine such as tectonic or XeLaTeX, not pdfLaTeX. `init` never overwrites an existing `template.tex`, so to get the fix in yours, replace the `fontenc` and `inputenc` lines with `\usepackage{fontspec}` and `\tracinglostchars=3`. ([#63](https://github.com/0xboris/invox/pull/63))
- **BREAKING:** `customer list`, `archive list` and `template list` escape backslash, tab, CR and newline inside fields when piped (`\\`, `\t`, `\r`, `\n`), so each record is one line. Scripts that read Windows paths from `template list` must unescape them. On a terminal, the lists print a header and aligned columns, and an empty list prints a hint on stderr. ([#76](https://github.com/0xboris/invox/pull/76))
- **BREAKING:** re-archiving a working copy from `invox archive edit` asks for confirmation on a terminal and needs `--yes` otherwise. Declining, or running without a terminal and without `--yes`, exits 2 and leaves the archive unchanged. ([#67](https://github.com/0xboris/invox/pull/67))
- **BREAKING:** `invox email -o PATH` refuses to overwrite an existing file unless you pass `--force`, and no longer deletes the draft after opening it. ([#61](https://github.com/0xboris/invox/pull/61))
- **BREAKING:** `invox new -e`, `invox config` and `invox customer config` exit 2 with a hint instead of starting an editor when stdin or stderr is not a terminal. The editor now runs directly from `VISUAL` or `EDITOR` instead of through `$SHELL -lc`, so editor settings that relied on aliases or functions from a login profile no longer resolve. ([#85](https://github.com/0xboris/invox/pull/85))
- **BREAKING:** `config.yaml` is strict. An unknown key such as `numbering.patern`, or a wrong type such as `archive.dir: 5`, is an error with file and line instead of being ignored. `numbering.start` must be a YAML integer, so `start: "3"` must become `start: 3`. ([#87](https://github.com/0xboris/invox/pull/87))
- **BREAKING:** on macOS, the default archive directory honours `XDG_DATA_HOME` when it is set. ([#87](https://github.com/0xboris/invox/pull/87))
- **BREAKING:** unknown keys in an invoice, `invoice_defaults.yaml`, `issuer.yaml` or the customer's entry in `customers.yaml` fail `validate`, `render`, `build` and `email` (and `new`, for defaults) with `file:line: unknown key` and a hint. Archived invoices still open with `archive edit` and `new --from-last`, which keep their keys. ([#89](https://github.com/0xboris/invox/pull/89))
- **BREAKING:** amounts, quantities and VAT rates must be plain decimals such as `12`, `12.50` or `-3.5`. Hex, octal prefixes, fractions (`1/3`), exponents (`1e3`) and `.5` are validation errors. ([#58](https://github.com/0xboris/invox/pull/58))
- **BREAKING:** a key defined twice in the same YAML mapping is an error with its file and line. Before, the last value won. A `<<` merge key whose value is not a mapping or a list of mappings is also an error. ([#66](https://github.com/0xboris/invox/pull/66))
- **BREAKING:** `numbering.pattern` is rejected when it has more than one `{counter}`, when `{customer_id}` or `{customer_code}` sits directly next to `{counter}`, when `{counter:WIDTH}` is above 20, or when it has invalid UTF-8. The error suggests a separated pattern such as `{customer_code}-{counter:04}`. After switching, set `numbering.start` to one more than the last number used, or the counter restarts. ([#70](https://github.com/0xboris/invox/pull/70))
- The module path is `github.com/0xboris/invox`, and building from source needs Go 1.24 or later. ([#55](https://github.com/0xboris/invox/pull/55), [#56](https://github.com/0xboris/invox/pull/56))
- Numeric YAML values used as text keep their written form, so `12.50` stays `12.50` instead of `12.5`. ([#58](https://github.com/0xboris/invox/pull/58))
- Unit prices with sub-cent precision show up to 4 decimals (for example `0,125`) instead of being rounded to cents. Totals are unchanged. ([#69](https://github.com/0xboris/invox/pull/69))
- When a command needs `config.yaml` and the file is broken, the error suggests running `invox config`. ([#64](https://github.com/0xboris/invox/pull/64))
- Validation reports invalid values in `customers.yaml`, `issuer.yaml` and invoice files with `file:line` and the field path, all in one run. It no longer lists every customer field as missing when the customer itself is missing or unknown. ([#77](https://github.com/0xboris/invox/pull/77), [#89](https://github.com/0xboris/invox/pull/89))
- The search for `customers.yaml`, `issuer.yaml`, `invoice_defaults.yaml` and the template stops at the nearest directory with `.git`, `invox.yaml` or `invoice_defaults.yaml`, and never enters `$HOME` unless it is the working directory. Relative `XDG_CONFIG_HOME`, `XDG_DATA_HOME` and `APPDATA` values are ignored. ([#87](https://github.com/0xboris/invox/pull/87))
- `invox email` without `-o` leaves the draft in its own temporary directory instead of deleting it a few seconds after opening. Each run removes `invox-email-*` directories older than 24 hours. ([#81](https://github.com/0xboris/invox/pull/81))
- `invox email` finds archived invoices the same way `invox archive list` does: it follows a symlinked `archive.dir` and skips `.history` backups. ([#98](https://github.com/0xboris/invox/pull/98))
- Ctrl-C while an editor runs goes to the editor only, and invox waits for it. A failing editor is named in the error with its exit status. ([#85](https://github.com/0xboris/invox/pull/85))
- Misspelt commands and flags suggest the closest name, for example `unknown flag: --nmaes; did you mean --names?`. `--` ends the flags on every command. ([#88](https://github.com/0xboris/invox/pull/88), [#90](https://github.com/0xboris/invox/pull/90), [#91](https://github.com/0xboris/invox/pull/91), [#92](https://github.com/0xboris/invox/pull/92), [#94](https://github.com/0xboris/invox/pull/94), [#95](https://github.com/0xboris/invox/pull/95), [#97](https://github.com/0xboris/invox/pull/97))
- Help pages are generated from the commands. Each page lists its flags, global flags and `$ invox` examples, nouns list their subcommands, and the root help groups commands and lists the help topics. ([#99](https://github.com/0xboris/invox/pull/99))
- `invox completion` adds bash, fish and powershell next to zsh, and the zsh script is now generated by cobra. The scripts complete commands, flags, file names, customer IDs, template names and archived invoices. ([#99](https://github.com/0xboris/invox/pull/99))
- `invox new` and `invox archive edit` name `--force` in the error when their output file already exists. ([#102](https://github.com/0xboris/invox/pull/102))

### Added

- `invox version` and `invox --version` print the version and build date. `go install github.com/0xboris/invox/cmd/invox@latest` installs invox. ([#56](https://github.com/0xboris/invox/pull/56))
- An MIT license. ([#60](https://github.com/0xboris/invox/pull/60))
- `--json <fields>` on `customer list`, `archive list`, `template list`, `validate`, `new`, `increment`, `render`, `build`, `archive` and `archive edit` prints one JSON document on stdout. Each command's help lists its fields under "JSON fields". ([#101](https://github.com/0xboris/invox/pull/101))
- `-n/--dry-run` on `new`, `increment`, `render`, `build`, `archive`, `archive edit` and `email` runs the same checks as a real run, prints `Would ...` lines on stderr and writes nothing. ([#102](https://github.com/0xboris/invox/pull/102))
- `--force` on `new` and `archive edit` replaces an existing output file. It never replaces a file inside `archive.dir`. ([#102](https://github.com/0xboris/invox/pull/102))
- `--yes` on `invox archive` and `invox build --archive` replaces an archived invoice without asking. The previous version is kept in `archive.dir/.history/`, and the replacement is reported on stderr. ([#67](https://github.com/0xboris/invox/pull/67))
- `--force` on `invox email` overwrites an existing `-o` file. ([#61](https://github.com/0xboris/invox/pull/61))
- Man pages in `share/man/man1/` and Markdown reference pages in `docs/cli/`, one per command and help topic. ([#99](https://github.com/0xboris/invox/pull/99))
- `invox help exit-codes` and `invox help environment`. ([#75](https://github.com/0xboris/invox/pull/75))
- Global `--config PATH` flag and `INVOX_CONFIG_DIR` variable to choose the config file or directory, and `invox config paths` to show which files invox uses. ([#87](https://github.com/0xboris/invox/pull/87))
- Global `--no-input` flag and `INVOX_PROMPT_DISABLED` variable. With either set, invox never prompts or opens an editor. ([#85](https://github.com/0xboris/invox/pull/85))
- Ctrl-C (SIGINT) stops the programs invox started, removes its temporary files and exits 130. SIGTERM does the same and exits 143. Ctrl-C at a confirmation prompt exits 2. ([#83](https://github.com/0xboris/invox/pull/83))
- `invox validate` warns on stderr when the invoice number is already archived under another file. ([#62](https://github.com/0xboris/invox/pull/62))
- `invox new` and `invox increment` warn on stderr when archived invoices of the customer don't match `numbering.pattern` and were left out of the next number. ([#84](https://github.com/0xboris/invox/pull/84))
- `invox init --force` copies missing files from the old `invoice-tool` config directory. On a terminal, `init` asks instead. ([#87](https://github.com/0xboris/invox/pull/87))
- Release archives for Linux, macOS and Windows on amd64 and arm64, with man pages and bash, zsh, fish and PowerShell completions, plus a Homebrew cask: `brew install 0xboris/tap/invox`. ([#52](https://github.com/0xboris/invox/issues/52))
- `make lint`, `make fmt` and `make tidy` targets. ([#60](https://github.com/0xboris/invox/pull/60))

### Deprecated

- Single-dash long flags such as `-names`, `-input`, `-config` or `-help` still work but print `warning: -NAME is deprecated; use --NAME` on stderr. Use the double-dash form. ([#88](https://github.com/0xboris/invox/pull/88), [#90](https://github.com/0xboris/invox/pull/90), [#91](https://github.com/0xboris/invox/pull/91), [#92](https://github.com/0xboris/invox/pull/92), [#94](https://github.com/0xboris/invox/pull/94), [#95](https://github.com/0xboris/invox/pull/95), [#97](https://github.com/0xboris/invox/pull/97))
- The legacy `invoice-tool` config directory still works, but a command that reads a file from it prints a warning on stderr. Run `invox init` to copy the files into the invox config directory. ([#87](https://github.com/0xboris/invox/pull/87))

### Removed

- **BREAKING:** the bundled Ubuntu fonts in `fonts/`. A template that needs fonts must ship them in its own template directory, which invox still copies. ([#60](https://github.com/0xboris/invox/pull/60))
- The `make init`, `validate`, `render`, `email`, `send`, `pdf` and `archive` wrappers. Run the matching `invox` command instead. ([#60](https://github.com/0xboris/invox/pull/60))
- invox no longer sets `INVOX_EDITOR` for the editor or reads `SHELL`. ([#85](https://github.com/0xboris/invox/pull/85))

### Fixed

- Numbers with a leading zero are no longer read as octal. `quantity: 010` means 10, and postal codes like `01067` and invoice numbers like `0042` keep their digits. ([#58](https://github.com/0xboris/invox/pull/58))
- `validate`, `render`, `build` and `email` reject a negative `invoice.paid_amount`. Before, such an invoice showed an inflated outstanding amount and an EPC QR code for more than the total. ([#57](https://github.com/0xboris/invox/pull/57))
- Rendering is deterministic. Text in invoice data that looks like a placeholder, such as `@@TOTAL@@` in a customer name, is printed as written. ([#59](https://github.com/0xboris/invox/pull/59))
- `invox new` no longer gives two drafts the same number. It counts invoices with `status: draft` or `status: built` in the current and output directories. ([#62](https://github.com/0xboris/invox/pull/62))
- `invox archive` and `invox build --archive` refuse an invoice whose number is already archived under another file. The error names both files and suggests `invox increment -i FILE`. ([#62](https://github.com/0xboris/invox/pull/62))
- A symlinked `archive.dir` is followed when invox looks up archived invoices. ([#62](https://github.com/0xboris/invox/pull/62))
- A broken `config.yaml` no longer breaks `invox init`, usage errors, or commands that get every path they need from flags. ([#64](https://github.com/0xboris/invox/pull/64))
- YAML merge keys (`<<: *anchor`, `<<: [*a, *b]`) are applied in every invox YAML file. Before, they were ignored, so a position that took its VAT rate from an anchor was billed at the invoice's rate. ([#66](https://github.com/0xboris/invox/pull/66))
- When invox rewrites an invoice, merge keys stay `<<: *anchor` instead of becoming `!!merge <<: *anchor`. Errors in Markdown archive front matter give the line number in the `.md` file. ([#66](https://github.com/0xboris/invox/pull/66))
- `invox build` no longer changes an archived invoice's status from `archived` to `built`. ([#67](https://github.com/0xboris/invox/pull/67))
- Invoice values that start with `[` or `*`, such as a period of `[Q1] 2026`, no longer break the TeX build or lose the `*`. ([#69](https://github.com/0xboris/invox/pull/69))
- Email drafts encode the UTF-8 body as quoted-printable, and attachment names with quotes or non-ASCII characters are encoded correctly. ([#69](https://github.com/0xboris/invox/pull/69))
- Amounts above 10,000,000,000,000 are rejected with an error that names the limit instead of wrapping to wrong or negative totals. IBANs with check digits 00, 01 or 99 are rejected. ([#70](https://github.com/0xboris/invox/pull/70))
- A YAML alias to a node that contains it no longer crashes invox, and aliases that expand to more than 100,000 nodes fail at once instead of hanging. Both errors name the file and line. ([#74](https://github.com/0xboris/invox/pull/74))
- `invox help config` shows the legacy config path under `$XDG_CONFIG_HOME` instead of a hard-coded `~/.config/invoice-tool` path. ([#75](https://github.com/0xboris/invox/pull/75))
- A mapping or list where text belongs is an error instead of being printed into the PDF as `map[...]`. `issuer.payment.due_days: missing value` is reported once instead of twice. ([#89](https://github.com/0xboris/invox/pull/89))
- `invox archive edit` working copies made on Windows now resolve on macOS and Linux. ([#55](https://github.com/0xboris/invox/pull/55))
- `new CUST-001 -o` and `build x.yaml -o` say `flag needs an argument: -o`, and a misspelt flag such as `--form-last` is reported as an unknown flag instead of as an unexpected argument. A blank `-o` counts as no `-o`. ([#92](https://github.com/0xboris/invox/pull/92), [#94](https://github.com/0xboris/invox/pull/94))
- `invox -version` prints the version, and `invox help --help` prints the root help, instead of failing with exit 2. ([#97](https://github.com/0xboris/invox/pull/97))
- `make build` always rebuilds and never leaves a stale binary. ([#60](https://github.com/0xboris/invox/pull/60))

### Security

- Files that hold bank or customer data (`issuer.yaml`, `customers.yaml`, archived invoices and their backups) are created readable only by you (0600), and new config and archive directories are 0700. Rewriting a file keeps its mode, and a symlinked invoice stays a symlink. ([#86](https://github.com/0xboris/invox/pull/86))
- List commands strip ANSI escape sequences, control characters and bidi and zero-width characters from YAML text before printing. ([#76](https://github.com/0xboris/invox/pull/76))

[Unreleased]: https://github.com/0xboris/invox/commits/main
