# Create and check an invoice

A user creates an invoice for a customer with the next free number, checks it, and renders it to LaTeX.

## Sub-features

- `new` writes `<invoice number>.yaml` in the work directory, with the number made from the customer ID and a counter.
- `validate` loads the invoice with its customer and issuer and prints the number, the customer, the line item count, and the total.
- `render` writes `invoice.tex`.

## How to get to it (user POV)

- `invox new CUST-001`, then `invox validate CUST-001-001.yaml`, then `invox render CUST-001-001.yaml`.

## Driving it with scripts/app

Preconditions:

- `$app doctor "$run"` reports `ok`.
- `$app run "$run" init init` exited with 0.

- **Create.** Run `$app run "$run" new new CUST-001`. The exit code is 0, stdout is `CUST-001-001.yaml`, and stderr is `Created CUST-001-001.yaml for CUST-001 (CUST-001-001)`.
- **Read back.** Run `$app show "$run" new-file CUST-001-001.yaml`. The file has `number: "CUST-001-001"` and `status: draft` under `invoice:`.
- **Validate.** Run `$app run "$run" validate validate CUST-001-001.yaml`. The exit code is 0, and stderr is `Validation OK: CUST-001-001 for CUST-001, 1 line item(s), total 120,00 €`.
- **Render.** Run `$app run "$run" render render CUST-001-001.yaml`. The exit code is 0, stdout is `invoice.tex`, and `$app show "$run" render-file invoice.tex` prints the LaTeX source.
- **Missing file.** Run `$app run "$run" validate-missing validate nope.yaml`. The exit code is 1, and stderr is `error: open nope.yaml: no such file or directory`.

## Gotchas

- The counter comes from the existing invoices and the archive, so a second `new CUST-001` in the same run creates `CUST-001-002.yaml`.
- Status lines go to stderr. Stdout holds only the path, so assert both.
