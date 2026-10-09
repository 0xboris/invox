# Output changes: U6b, typed errors for missing files and failed programs

## A paths.* setting that names a missing file

A `paths.customers`, `paths.issuer`, `paths.defaults` or `paths.template` setting in
config.yaml that names a file that does not exist is a problem with config.yaml. The store
returns a `*billing.ConfigError` wrapping a `*billing.FileNotFoundError` whose `Config` is the
config file. The message names the missing file and the setting, and `errorHint` adds the
`invox config` hint it gives every config error (with `--config <file>` when one was given).
Exit code 1, before and after. `customer edit` still opens the editor on the missing file.

`template list --json` reports the same error instead of an empty `default`: the field names
the template `render` and `build` use, and they fail.

| Command | Before | Now |
| --- | --- | --- |
| `invox customer list` with `paths.customers: nope.yaml` | `error: open /cfg/nope.yaml: no such file or directory` | `error: customers file /cfg/nope.yaml does not exist; paths.customers in /cfg/config.yaml sets it` and `Run 'invox config' to open and fix the config file.` |
| `invox validate inv.yaml -c customers.yaml` with `paths.issuer: nope.yaml` | `error: open /cfg/nope.yaml: no such file or directory` | `error: issuer file /cfg/nope.yaml does not exist; paths.issuer in /cfg/config.yaml sets it` and the hint |
| `invox new CUST-001 ...` with `paths.defaults: nope.yaml` | `error: open /cfg/nope.yaml: no such file or directory` | `error: defaults file /cfg/nope.yaml does not exist; paths.defaults in /cfg/config.yaml sets it` and the hint |
| `invox render inv.yaml ...` with `paths.template: nope.tex` | `error: open /cfg/nope.tex: no such file or directory` | `error: template file /cfg/nope.tex does not exist; paths.template in /cfg/config.yaml sets it` and the hint |
| `invox template list --json name,default` with `paths.template: nope.tex` | exit 0, `[]` (no default) | exit 1, the template row's error and hint |
| `invox --config c.yaml validate -i inv.yaml` with `paths.issuer: nope.yaml` in `c.yaml` | `error: open nope.yaml: no such file or directory` | `error: issuer file nope.yaml does not exist; paths.issuer in c.yaml sets it` and `Run 'invox --config c.yaml config' to open and fix the config file.` |

Changed files:

- `cmd/invox/testdata/script/broken_config.txtar`: a new case, `missing-paths.yaml` and
  `want-missing-paths.txt` pin the last row.
