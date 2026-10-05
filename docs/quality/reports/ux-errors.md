# invox audit: errors, exit codes, help, flags, grammar

## (a) Verdict

invox's commands are consistent about the 0/1/2 split in the common cases, `-h` works at every level, and a few messages are genuinely actionable. Underneath, exit codes are about 45 hand-returned ints across 7 files, and printing is spread over 4 usage printers and dozens of `Fprintln(os.Stderr, err)` calls. Usage errors become exit 1 when the config is broken. The custom `reorderArgs` produces misleading errors (a dangling flag swallows the positional, unknown flags are reported as "unexpected arguments", `--` is ignored). Help is 849 hand-written lines and has already drifted from the spec, completion and README. `version`, `help exit-codes` and `help environment` don't exist, and nothing has `--yes`, `--force` or `--dry-run`.

## (b) Observed exit codes

| Scenario | stdout | stderr | code | Expected per skill |
|---|---|---|---|---|
| `new CUST-001` (success) | `Created …` | – | 0 | 0 ✓ |
| `invox` (no args) | – | `error: missing subcommand` + full root help (80 lines) | 2 | usage code + short hint |
| `frobnicate` | – | `error: unknown subcommand` + 80 lines, no suggestion | 2 | usage + "did you mean" |
| `validate --bogus` | – | `flag provided but not defined: -bogus` + 10-line usage | 2 | usage ✓ (text rewrites `--` to `-`) |
| `new CUST-001 --bogus` | – | `unexpected arguments: --bogus` | 2 | should say "unknown flag" |
| `new` (missing arg) | – | `missing required arguments: CUSTOMER_ID` | 2 | ✓ |
| `new CUST-001 extra` | – | `unexpected arguments: extra` | 2 | ✓ |
| `new --bogus` with broken config.yaml | – | `config.yaml: yaml: line 1: …` | **1** | usage code: parsing must come before config |
| `validate -i missing.yaml` | – | `open /abs/…/missing.yaml: no such file or directory` | 1 | 1 ✓, message is raw |
| invalid YAML | – | `/abs/bad.yaml: yaml: line 1: …` | 1 | 1 ✓ |
| validation failure | – | one problem per line, no header | 1 | 1 ✓ |
| `build` without tectonic | – | `tectonic not found in PATH` + `brew install` hint | 1 | 1 ✓ (hint is macOS-only) |
| `email` on a draft | – | `invoice.status must be built or archived …` | 1 | 1 ✓ |
| `--help`, `-h`, `help new`, `archive list -h`, `customer list --help`, `help archive edit` | help | – | 0 | 0 ✓ |
| `help nope` | – | error + 80-line root help | 2 | usage |
| `--version` / `version` | – | `unknown subcommand` + 80 lines | 2 | 0 + version |
| `help exit-codes` / `help environment` | – | unknown help topic | 2 | topic page, 0 |
| `completion bash` | – | `unsupported shell "bash"` | 2 | – |
| `--help`, `new --help`, `completion zsh` with broken config | help | – | 0 | 0 ✓ |

Exit 2 for usage errors isn't documented in the help or the README, but 14 tests assert it.

## (c) Findings

**F1 [major] There's no single mapping from errors to exit codes, and none of it is documented.** VERIFIED+READ. Rule: "Commands return errors; one place maps them to exit codes… Document them in `help exit-codes`." The command files contain 29 × `return 1` and 9 × `return 2`, and `parsing.go` adds 8 more coded returns. Usage text is printed by `rootUsageError`, `customerUsageError`, `templateUsageError` and `printCommandError` (cli.go:144-160, help.go:837). `email` returns 2 for a path-resolution failure (commands_invoice.go:183). **Fix:** have every `runX` return `error`, with typed `FlagError` and `SilentError`. One `Main()` turns those into codes and prints once. Keep 2 for usage errors, because tests (and maybe scripts) depend on it, and add `invox help exit-codes`: 0 ok, 1 error, 2 usage.

**F2 [major] The exit code of a usage error depends on the config.** VERIFIED. `parseCommand` calls `invoice.DefaultOptions()`, which parses config.yaml and resolves paths, *before* `fs.Parse` (parsing.go:58). With a broken config, `invox new --bogus` exits 1 with a YAML error. Even fully explicit `validate -i x -c C -u U` fails on the unrelated config. **Fix:** parse flags first, then resolve defaults lazily, and only for paths the flags didn't supply.

**F3 [major] `reorderArgs` produces wrong or misleading errors.** VERIFIED.
- A dangling value flag swallows the positional. `new CUST-001 -o` gives `missing required arguments: CUSTOMER_ID`, and `email snap.yaml --to` gives `missing required input: INVOICE.yaml, INVOICE.pdf, or -i, --input`.
- An unknown flag after a positional is reported as a positional: `new CUST-001 --form-last` gives `unexpected arguments: --form-last`, with no suggestion.
- Single-dash long flags only work before the positional. `new -customers C CUST-001` works, but `new CUST-001 -customers C` and `-output=x` fail.
- `--` isn't honoured: `new CUST-001 -- -o d5.yaml` gives `unexpected arguments: --`.
- Only new, build and email reorder at all. In `archive CUST.yaml -i x.yaml` the flags after the positional are ignored.

**Fix:** move to pflag/cobra, or a single parser driven by the spec. Reject a missing value as `flag needs an argument: --to`. Echo unknown flags as the user typed them and add a "did you mean" (edit distance 2).

**F4 [major] Positional input is inconsistent across commands.** VERIFIED. `build`, `email` and `archive` accept `INVOICE.yaml`. `validate`, `render` and `increment` reject it with `unexpected arguments: CUST-001-001.yaml`, and the message doesn't mention `-i`. The zsh completion offers a positional for `render` anyway (commands_template.go, render case). **Fix:** accept the positional everywhere. The change is additive, so it doesn't break any script.

**F5 [major] Runtime error messages are low quality.** VERIFIED. Rules: "what happened → why → what to do", one consistent shape. Usage errors start with `error:` but runtime errors have no prefix. Messages print absolute paths and raw Go errors. The worst 5:
1. `new CUST-001 -o` → `missing required arguments: CUSTOMER_ID` (the argument was given)
2. `email snap.yaml --to` → `missing required input …` (the input was given)
3. `new --bogus` with a bad config → `…/config.yaml: yaml: line 1: did not find expected ',' or ']'`, exit 1, with no hint to run `invox config`
4. `validate -i v1.yaml` with an unknown customer → `unknown customer_id NOPE` followed by 7 spurious `customer.name: missing value` lines, and no hint to run `invox customer list`
5. `render CUST.yaml` → `unexpected arguments: CUST-001-001.yaml` (no hint to use `-i`). Also `archive d.yaml` → `…/invoices/d.yaml already exists`, with no next step.

The best 3:
- `keep.yaml already exists; choose a different -o/--output path`
- ``template "nosuch" not found; run `invox template list` ``
- `customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at …` (READ, parsing.go:174)

**Fix:** print through one path with an `error: ` prefix (or none) used everywhere, show display-relative paths, wrap OS errors (`cannot read invoice x.yaml: file not found`), stop after an unknown customer, and add next-step commands (`run invox build first`). Make the tectonic hint platform-aware.

**F6 [minor] Usage errors are noisy.** VERIFIED. Rule: terse usage only. An unknown root subcommand dumps 80 lines. `customer frob` dumps 82, including the YAML schema and an example. `template frob` dumps 102. The per-command `printCommandError` (10 lines plus `Use 'invox new --help'`) is the right model. **Fix:** print `error: …` plus `Run 'invox --help' for usage.` and add subcommand suggestions.

**F7 [major] `version`/`--version` and the help topics are missing.** VERIFIED. Both version forms exit 2. Env vars (XDG_CONFIG_HOME, XDG_DATA_HOME, APPDATA, VISUAL, EDITOR, SHELL) and the precedence order aren't gathered in one place: precedence only appears in `help config`. **Fix:** add a `version` command and `--version` flag (version set with ldflags), plus `help environment` and `help exit-codes`.

**F8 [major] Help is hand-written and has drifted.** VERIFIED+READ. Rule: "Help is data (`Short`, `Long`, `Example` with `$ tool …`)… generate it so it can't drift."
- help.go is 849 lines of `Fprintf`. The flags in `printCommandHelp` are a second copy of `bindCommandFlags`.
- `invox archive --help` doesn't mention its own `edit` and `list` subverbs.
- Root "Optional flags" omits `-i/--input` and `--names`.
- `-t` is documented as "Path to invoice_template.tex", but the default file is `template.tex` and the flag also takes template names.
- The `send` alias appears only in tests.
- Summaries end with a period. Examples lack `$ ` and `#` comments.
- `help template` and `help config` act as both a command page and a topic page, while `help customer` and `help customers` are different pages.
- `-o` rejects `.yml` even though `.yml` input is accepted.

**Fix:** extend `commandSpec` with Long, Examples, flags and Hidden, and render help, completion and the README table from it. Add a drift test.

**F9 [minor] Completion is zsh-only and hand-maintained.** VERIFIED. It completes arguments for only template, completion, render and build. Everything else (new, email, archive edit/list, customer, validate, increment) falls through to `_files`. Template-name completion fails silently when the config is broken. **Fix:** generate completion from the spec, and add bash, fish and powershell (cobra provides all four).

**F10 [major] Destructive and mutating operations have no confirmation or preview.** VERIFIED. Rule: `--yes`, `--force` and `--dry-run` are distinct flags, and hard-to-undo changes need confirmation.
- `new` safely refuses to overwrite (good), but there's no `--force`.
- `increment` rewrites the file in place with no `--dry-run`.
- `archive c.yaml` on an `editing` working copy silently replaced the archived invoice.
- Two different invoices with the same number `CUST-001-001` (c.yaml and d.yaml) were both archived without a warning. That's a domain issue for the numbering audit.
- `build --archive` moves files.
- All of `--yes`, `--force` and `--dry-run` give `unexpected arguments`.

**Fix:** `archive` asks before replacing an existing entry on a TTY and requires `--yes` otherwise. `-n/--dry-run` on `increment`, `archive` and `new` prints the target and number without writing. `--force` on `new` overwrites.

**F11 [minor] tectonic writes to invox's stdout.** READ, service.go:941. `cmd.Stdout = os.Stdout`, without a context. Child output should go to stderr, so stdout carries only data such as `Built x.pdf`. Use `exec.CommandContext`.

**F12 [minor] Flag letters are odd.** READ.
- `-s/--source` points at the defaults file (config key `paths.defaults`), and `-s` conventionally means `--state`.
- `-u` for issuer isn't mnemonic.
- Siblings are otherwise consistent (`-c` and `-u` mean the same everywhere).
- Success output leaks LaTeX: `total 120,00 \euro`. VERIFIED.

**Fix:** add `--defaults` as the canonical name and keep `--source`/`-s` as hidden aliases. Keep `-u`.

## (d) Target command tree

```
invox init
invox config [edit]                 # bare `config` keeps working
invox customer list | edit          # `customer config` → hidden alias + warning
invox template list [--names]
invox new CUSTOMER_ID [-o] [--defaults|-s] [--from-last] [-e] [--force] [-n]
invox increment   [INVOICE] [-i] [-n]
invox validate    [INVOICE] [-i]
invox render      [INVOICE] [-i] [-o] [-t NAME|PATH]
invox build       [INVOICE] [-i] [-o] [-t] [--archive]
invox email       [INVOICE|PDF] [-i] [-p] [-o] [--to] [--subject]   # `send` → hidden, warns "does not send"
invox archive add  INVOICE [--yes] [-n]   # bare `archive FILE` → hidden form, warns
invox archive list
invox archive edit NAME             # optionally alias `archive restore`
invox completion zsh|bash|fish|powershell
invox version   (+ --version)
invox help [exit-codes|environment|config|customers|issuer|defaults|template]
```

Since invox has only one primary noun, keep the invoice verbs at the top level, grouped as "Invoice commands" in help, rather than moving them under `invox invoice …`. That move would break every script for no real gain.

The real grammar defect is `archive`, which is both a verb (`archive FILE`) and a noun (`archive list`). A file named `list` or `edit` can't be archived, and `archive --help` hides the subverbs.

Migration (per the script-facing contract rule):
- Release N adds `archive add`, `customer edit`, `config edit`, `--defaults`, `version`, the help topics and positional input everywhere. All of these are additive.
- The old forms (`archive FILE`, `customer config`, `send`, `--source`) keep working, hidden from help, and each prints one stderr line: ``warning: `invox archive FILE` is deprecated; use `invox archive add FILE` ``.
- Remove them no earlier than N+1, and record them in the changelog.
- Keep exit code 2 for usage errors and document it, rather than switching to gh's 1.
