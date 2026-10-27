# Set up and customers

A user creates the config directory with its starter files, lists the customers in `customers.yaml`, and opens that file in an editor.

## Sub-features

- `init` writes `config.yaml`, `customers.yaml`, `issuer.yaml`, `invoice_defaults.yaml`, and `template.tex` into the config directory.
- `customer list` prints one tab-separated row per customer: ID, name, and status.
- `customer edit` refuses to start an editor without a terminal, names the file to edit instead, and exits with 2.

## How to get to it (user POV)

- `invox init`, then `invox customer list`, then `invox customer edit`.

## Driving it with scripts/app

Preconditions:

- `$app doctor "$run"` reports `ok`.

- **Init.** Run `$app run "$run" init init`. The exit code is 0, and stderr has `Initialized <config directory>` and one `created <file>` line for each of the five files.
- **List.** Run `$app run "$run" customer-list customer list`. The exit code is 0, and stdout is `CUST-001	Appsters GmbH	active`, the starter customer.
- **Editor gate.** Run `$app run "$run" customer-edit customer edit`. The exit code is 2, and stderr starts with `error: cannot open an editor: prompts are disabled` and ends the line with `edit <config directory>/customers.yaml directly`.

## Gotchas

- `init` on a config directory that already has the files leaves them unchanged and prints `exists <file>` for each one. Start a new run to see the `created` lines again.
- The run disables prompts, so no command in it ever starts an editor.
