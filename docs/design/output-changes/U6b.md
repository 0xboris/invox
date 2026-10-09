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

## validate --json prints no document for a support file that does not exist

`validate --json` prints a document only for an invoice with problems in its content; a file
that cannot be read prints none. Since U6, a missing `-c`, `-u` or `--defaults` file (and,
since the section above, a missing `paths.*` file) is a `*billing.FileNotFoundError` instead
of the OS's `*fs.PathError`, and `validate --json` printed `{"valid":false}` for it. It counts
as a file that cannot be read again. Exit code 1, before and after.

| Command | Before | Now |
| --- | --- | --- |
| `invox validate -i inv.yaml -c missing.yaml --json valid` | stdout `{"valid":false}`, stderr `error: customers file missing.yaml does not exist` | no stdout, the same stderr |

Changed files:

- `cmd/invox/testdata/script/json.txtar`: a new case pins the row.

## An invoice argument that does not exist

`billing.File` gains `InvoiceFile`, and the store's `Load` and `Update` (through `Rewrite`)
return a `*billing.FileNotFoundError` of it for an invoice file that does not exist. Every verb
that reads an invoice argument words it the same way. It stays a runtime error (exit 1), like
a missing support file named on the command line.

| Command | Before | Now |
| --- | --- | --- |
| `invox validate /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |
| `invox render /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |
| `invox build -i /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |
| `invox email /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |
| `invox increment /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |
| `invox archive add /nope.yaml` | `error: open /nope.yaml: no such file or directory` | `error: invoice file /nope.yaml does not exist` |

Changed files (each pins its command's row with a `missing.yaml` case; `validate`,
`increment` and `json` had one matching only `missing\.yaml`):

- `cmd/invox/testdata/script/validate.txtar`
- `cmd/invox/testdata/script/render.txtar`
- `cmd/invox/testdata/script/build.txtar`
- `cmd/invox/testdata/script/email.txtar`
- `cmd/invox/testdata/script/increment.txtar`
- `cmd/invox/testdata/script/archive.txtar`
- `cmd/invox/testdata/script/json.txtar`

## Apple Mail failures are a billing.ToolFailedError

`applemail.Composer.Draft` returns a `*billing.ToolFailedError` for osascript when the
program fails, like tectonic and the editor, instead of wrapping the bare exit status. Exit
code 1, before and after. osascript's own output still comes first on stderr.

| Command | Before | Now |
| --- | --- | --- |
| `invox email inv.yaml` on macOS, osascript exits 1 | `error: failed to open editable email draft: exit status 1` | `error: osascript exited with status 1` |

Changed files:

- `cmd/invox/testdata/script/email_macos.txtar`: a new case and `want-osascript-failed.txt`
  pin the row. `FAKE_OPEN_FAIL` now makes the fake osascript fail too.

## The OS opener's failures name the program

`opener.Opener.Open` returns a `*billing.ToolMissingError` when `open`, `xdg-open` or `cmd`
is not on PATH, and a `*billing.ToolFailedError` when it fails, instead of the bare exec
error. The email mailer still says what it did with the draft first. Exit code 1, before and
after; the missing opener has no hint line.

| Command | Before | Now |
| --- | --- | --- |
| `invox email inv.yaml -o d.eml`, xdg-open exits 1 | `error: created d.eml but failed to open it: exit status 1` | `error: created d.eml but failed to open it: xdg-open exited with status 1` (`open` on macOS, `cmd` on Windows) |
| `invox email inv.yaml -o d.eml`, xdg-open not installed | `error: created d.eml but failed to open it: exec: "xdg-open": executable file not found in $PATH` | `error: created d.eml but failed to open it: xdg-open not found in PATH` |
| `invox email inv.yaml`, xdg-open not installed | `error: failed to open email draft: exec: "xdg-open": executable file not found in $PATH` | `error: failed to open email draft: xdg-open not found in PATH` |

Changed files:

- `cmd/invox/testdata/script/email.txtar`: `want-open-failed.txt` names the opener through
  `$OPENER` (compared with `cmpenv`), and a new case with an empty PATH and
  `want-no-opener.txt` pins the second row.

## An editor that is not installed

`editor.Editor.Edit` returns a `*billing.ToolMissingError` naming the editor setting when the
program is not on PATH, with a hint, instead of the exec error. Exit code 1, before and after.
No script reaches the editor (it needs a terminal); `TestConfigReportsAnEditorThatIsNotInstalled`
pins the row.

| Command | Before | Now |
| --- | --- | --- |
| `VISUAL=nano invox config` on a terminal, nano not installed | `error: failed to open /cfg/config.yaml: exec: "nano": executable file not found in $PATH` | `error: failed to open /cfg/config.yaml: editor "nano" not found in PATH` and `Set VISUAL or EDITOR to an installed editor, then rerun this command.` |

The same holds for the other commands that open the editor, `new -e` and `customer edit`.

No testscript golden, `docs/cli` or `share/man` page changed.
