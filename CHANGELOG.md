# Changelog

All notable changes to invox are documented in this file. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Unreleased

### Added

- `invox archive add INVOICE` archives an invoice. It takes every flag that
  `invox archive INVOICE` took: `-i, --input`, `--yes`, `-n, --dry-run` and
  `--json`. A file named `list` or `edit` can now be archived.
- `invox config edit` opens config.yaml, as `invox config` does.
- `invox new --defaults FILE` names the invoice_defaults.yaml file.
- `increment`, `validate` and `render` take the invoice as an `INVOICE`
  argument, as `build`, `email` and `archive add` do.

### Changed

- Every command that takes an `INVOICE` argument and `-i, --input` accepts
  both when they name the same file. When they name different files, the
  command exits 2 with a usage error.
- Without an invoice, `increment`, `validate` and `render` report
  `missing required input: INVOICE.yaml or -i, --input`, as `build` does.
- Help, the generated docs and shell completion list only the new forms
  below. `invox email --help` no longer lists the `send` alias.

### Deprecated

Each deprecated form still works and prints one warning line on stderr. They
will be removed no earlier than the next minor release.

- `invox archive INVOICE`: use `invox archive add INVOICE`.
- `invox customer config`: use `invox customer edit`.
- `invox new -s FILE` and `--source FILE`: use `--defaults FILE`.
- `invox send`: use `invox email`. invox only drafts an email.
- Single-dash long flags such as `-names`: use `--names`.
