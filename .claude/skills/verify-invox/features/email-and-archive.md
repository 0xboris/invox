# Email and archive

A user writes an email draft for a built invoice and opens it in the mail app, then archives the invoice and lists the archive.

## Sub-features

- `email -o draft.eml` writes an unsent `.eml` draft with the PDF attached and opens it with the platform's opener.
- `email` refuses an invoice whose status is not `built` or `archived`.
- `archive add` moves the invoice into the archive directory and prints its new path.
- `archive list` prints one row per archived invoice: file, customer, date, and status.

## How to get to it (user POV)

- `invox email CUST-001-001.yaml -o draft.eml`, then `invox archive add CUST-001-001.yaml`, then `invox archive list`.

## Driving it with scripts/app

Preconditions:

- The steps of [Create and check an invoice](./create-invoice.md) up to **Create** ran in this run.

- **Not built.** Before building, run `$app run "$run" email-draft-status email CUST-001-001.yaml -o draft.eml`. The exit code is 1, and stderr ends with ``invoice.status must be `built` or `archived` before creating an email draft, got `draft` ``.
- **Build.** Follow **Build** in [Build an invoice](./build-invoice.md).
- **Draft.** Run `$app run "$run" email email CUST-001-001.yaml -o draft.eml`. The exit code is 0, stdout is `draft.eml`, and stderr is `Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example`.
- **Read back.** Run `$app show "$run" email-file draft.eml`. The headers have `To: <office@appsters.example>`, `Subject: Invoice CUST-001-001`, and `X-Unsent: 1`. Run `$app show "$run" email-opened ../opened.log`. On Linux it has one line, `xdg-open <work directory>/draft.eml`.
- **Archive.** Run `$app run "$run" archive-add archive add CUST-001-001.yaml`. The exit code is 0, and stdout is the archived path, which ends with `invox/invoices/CUST-001-001.yaml`.
- **List.** Run `$app run "$run" archive-list archive list`. Stdout has the row `CUST-001-001.yaml	CUST-001	<today>	archived`.

## Gotchas

- Without `-o`, invox writes the draft into a new temporary directory, and on macOS it hands the message to Apple Mail through `osascript` instead. Both reach `opened.log` too, but `-o` keeps the draft in the work directory where `show` can read it.
- The archive directory sits under the run's home, so `stop` removes it with the run.
