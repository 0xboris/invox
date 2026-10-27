# Build an invoice

A user builds the PDF of an invoice with tectonic, and invox marks the invoice as built.

## Sub-features

- `build` writes `<invoice>.pdf` next to the invoice and sets `invoice.status` to `built`.
- Without `tectonic` on PATH, `build` exits with 1 and names where to install it.

## How to get to it (user POV)

- `invox build CUST-001-001.yaml`.

## Driving it with scripts/app

Preconditions:

- `$app doctor "$run"` reports `ok` with `tectonic` real or fake. With `tectonic=missing`, only the last step can run.
- The steps of [Create and check an invoice](./create-invoice.md) up to **Create** ran in this run.

- **Build.** Run `$app run "$run" build build CUST-001-001.yaml`. The exit code is 0, stdout is `CUST-001-001.pdf`, and stderr is `Built CUST-001-001.pdf for CUST-001 (CUST-001-001)`.
- **Read back.** Run `$app show "$run" build-status CUST-001-001.yaml`. The invoice now has `status: built`. With a real tectonic, `$app show "$run" build-pdf CUST-001-001.pdf` starts with `%PDF`.
- **No tectonic.** In a run started without `--fake-tectonic` on a machine without tectonic, run `$app run "$run" build-missing build CUST-001-001.yaml`. The exit code is 1, and stderr is `error: tectonic not found in PATH` followed by `Install it from https://tectonic-typesetting.github.io, then rerun this command.`

## Gotchas

- The fake tectonic proves invox's side only. A change to the template or the LaTeX output needs a run with the real tectonic.
- A failed build leaves `invoice.status` unchanged.
